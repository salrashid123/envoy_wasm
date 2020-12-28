// NOLINT(namespace-envoy)
#include <string>
#include <string_view>
#include <unordered_map>

#include "proxy_wasm_intrinsics.h"
#include "proxy_wasm_intrinsics_lite.pb.h"

#include "google/protobuf/util/json_util.h"

#include "examples/wasm-cc/echo/echo.pb.h"

static constexpr char EchoServerServiceName[] = "echo.EchoServer";
static constexpr char SayHelloMethodName[] = "SayHello";

using google::protobuf::util::JsonParseOptions;
using google::protobuf::util::error::Code;
using google::protobuf::util::Status;

using echo::EchoRequest;
using echo::EchoReply;
using echo::Config;

class ExampleRootContext : public RootContext {
public:
  explicit ExampleRootContext(uint32_t id, std::string_view root_id) : RootContext(id, root_id) {}

  bool onStart(size_t) override;
  bool onConfigure(size_t) override;
  void onTick() override;

   echo::Config config_;
};

class ExampleContext : public Context {
public:
  explicit ExampleContext(uint32_t id, RootContext* root) : Context(id, root) {}

  void onCreate() override;
  FilterHeadersStatus onRequestHeaders(uint32_t headers, bool end_of_stream) override;
  FilterDataStatus onRequestBody(size_t body_buffer_length, bool end_of_stream) override;
  FilterHeadersStatus onResponseHeaders(uint32_t headers, bool end_of_stream) override;
  FilterDataStatus onResponseBody(size_t body_buffer_length, bool end_of_stream) override;
  void onDone() override;
  void onLog() override;
  void onDelete() override;
};

class MyGrpcCallHandler : public GrpcCallHandler<google::protobuf::Value> {
 public:
  MyGrpcCallHandler(ExampleContext *context) { context_ = context;  }

  void onSuccess(size_t body_size) override { 
    LOG_INFO("GRPC call SUCCESS");
    WasmDataPtr response_data = getBufferBytes(WasmBufferType::GrpcReceiveBuffer, 0, body_size);
    const EchoReply& response = response_data->proto<EchoReply>();
    LOG_INFO("got gRPC Response: " + response.message());

    context_->setEffectiveContext(); 

    auto res = addRequestHeader("isAdmin", response.message());
    if (res != WasmResult::Ok) {
      LOG_ERROR("Modifying Header data failed: " + toString(res));
    }
    continueRequest();  
  }
  void onFailure(GrpcStatus status) override {
    LOG_INFO(" GRPC call FAILURE ");
    auto p = getStatus();
    LOG_DEBUG(std::string("failure ") + std::to_string(static_cast<int>(status)) +
             std::string(p.second->view()));
    context_->setEffectiveContext();              
    closeRequest();
  }

 private:
  ExampleContext *context_;

};


static RegisterContextFactory register_ExampleContext(CONTEXT_FACTORY(ExampleContext),
                                                      ROOT_FACTORY(ExampleRootContext),
                                                      "my_root_id");

bool ExampleRootContext::onStart(size_t) {
  LOG_INFO("onStart");
  return true;
}

bool ExampleRootContext::onConfigure(size_t config_size) {
  LOG_INFO("onConfigure called");
  proxy_set_tick_period_milliseconds(1000); // 1 sec
  const WasmDataPtr configuration = getBufferBytes(WasmBufferType::PluginConfiguration, 0, config_size);

    JsonParseOptions json_options;
    const Status options_status = JsonStringToMessage(
        configuration->toString(),
        &config_,
        json_options);
    if (options_status != Status::OK) {
      LOG_WARN("Cannot parse plugin configuration JSON string: " + configuration->toString());
      return false;
    }
    LOG_INFO("Loading Config: " + config_.clustername());
  return true;
}

void ExampleRootContext::onTick() { LOG_TRACE("onTick"); }

void ExampleContext::onCreate() { LOG_INFO(std::string("onCreate " + std::to_string(id()))); }

FilterHeadersStatus ExampleContext::onRequestHeaders(uint32_t, bool) {
  LOG_INFO(std::string("onRequestHeaders called ") + std::to_string(id()));
  auto result = getRequestHeaderPairs();
  auto pairs = result->pairs();
  LOG_INFO(std::string("headers: ") + std::to_string(pairs.size()));
  for (auto& p : pairs) {
    LOG_INFO(std::string(p.first) + std::string(" -> ") + std::string(p.second));
  }

  std::string jwt_string;
  if (!getValue(
          {"metadata", "filter_metadata", "envoy.filters.http.jwt_authn", "my_payload", "sub"}, &jwt_string)) {
    LOG_ERROR(std::string("filter_metadata Error ") + std::to_string(id()));
  }

  LOG_INFO(">>>>>>>>>>>>>  Calling GRPC for sub:" + jwt_string);
  ExampleRootContext *a = dynamic_cast<ExampleRootContext*>(root());
  GrpcService grpc_service;
  grpc_service.mutable_envoy_grpc()->set_cluster_name(a->config_.clustername());  
  std::string grpc_service_string;
  grpc_service.SerializeToString(&grpc_service_string);

  EchoRequest request;
  request.set_name(jwt_string);
  std::string st2r = request.SerializeAsString();
  HeaderStringPairs initial_metadata;
  initial_metadata.push_back(std::pair("parent", "bar"));
  auto res =  root()->grpcCallHandler(grpc_service_string, EchoServerServiceName, SayHelloMethodName, initial_metadata, st2r, 1000,
                              std::unique_ptr<GrpcCallHandlerBase>(new MyGrpcCallHandler(this)));

  if (res != WasmResult::Ok) {
    LOG_ERROR("Calling gRPC server failed: " + toString(res));
  }                         

  return FilterHeadersStatus::StopIteration;

  //addRequestHeader("fromenvoy", "newheadervalue");
  //return FilterHeadersStatus::Continue;
}

FilterHeadersStatus ExampleContext::onResponseHeaders(uint32_t, bool) {
  LOG_INFO(std::string("onResponseHeaders called ") + std::to_string(id()));
  auto result = getResponseHeaderPairs();
  auto pairs = result->pairs();
  LOG_INFO(std::string("headers: ") + std::to_string(pairs.size()));
  for (auto& p : pairs) {
    LOG_INFO(std::string(p.first) + std::string(" -> ") + std::string(p.second));
  }
  addResponseHeader("X-Wasm-custom", "FOO");
  replaceResponseHeader("content-type", "text/plain; charset=utf-8");
  removeResponseHeader("content-length");
  return FilterHeadersStatus::Continue;
}

FilterDataStatus ExampleContext::onRequestBody(size_t body_buffer_length,
                                               bool /* end_of_stream */) {
  auto body = getBufferBytes(WasmBufferType::HttpRequestBody, 0, body_buffer_length);
  LOG_INFO(std::string("onRequestBody ") + std::string(body->view()));
  return FilterDataStatus::Continue;
}

FilterDataStatus ExampleContext::onResponseBody(size_t /* body_buffer_length */,
                                                bool /* end_of_stream */) {
  setBuffer(WasmBufferType::HttpResponseBody, 0, 12, "Hello, world");
  return FilterDataStatus::Continue;
}

void ExampleContext::onDone() { LOG_WARN(std::string("onDone " + std::to_string(id()))); }

void ExampleContext::onLog() { LOG_WARN(std::string("onLog " + std::to_string(id()))); }

void ExampleContext::onDelete() { LOG_WARN(std::string("onDelete " + std::to_string(id()))); }