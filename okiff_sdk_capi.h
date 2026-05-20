#pragma once

#ifdef __cplusplus
extern "C" {
#endif


// Opaque handle to the SDK instance
typedef void* OkiffHandle;


// Callback types
typedef void (*OkiffMessageCallback)(const char* topic, const char* payload, void* userdata);
typedef void (*OkiffConnectionCallback)(int connected, int rc, void* userdata);


// Lifecycle
OkiffHandle okiff_create(void);
void        okiff_destroy(OkiffHandle handle);


// Initialization
int  okiff_init(OkiffHandle handle,
                const char* clientId,
                const char* brokerHost,
                const char* protocol,
                const char* username,
                const char* password,
                int cleanSession,
                int resumeSubs,
                int connectRetry,
                int autoReconnect,
                int connectRetryInterval,
                int maxReconnectInterval,
                int keepAlive,
                int pingTimeout,
                int writeTimeout,
                int orderMatters,
                int connectTimeout);

                
// Connection
int  okiff_connect(OkiffHandle handle);
void okiff_disconnect(OkiffHandle handle);
void okiff_stop(OkiffHandle handle);
int  okiff_is_connected(OkiffHandle handle);


// Pub/sub
void okiff_publish(OkiffHandle handle,
                   const char* topic,
                   const char* payload,
                   int qos,
                   int retained);


int  okiff_subscribe(OkiffHandle handle, const char* topic, int qos);
void okiff_unsubscribe(OkiffHandle handle, const char* topic);


// Callbacks
void okiff_set_message_callback(OkiffHandle handle, OkiffMessageCallback cb, void* userdata);
void okiff_set_connection_callback(OkiffHandle handle, OkiffConnectionCallback cb, void* userdata);


#ifdef __cplusplus
}
#endif