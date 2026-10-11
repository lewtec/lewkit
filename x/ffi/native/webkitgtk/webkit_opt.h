/* Optional WebKitGTK entry points. Missing symbols stay nil. */
uintptr_t webkit_network_session_new(char *dataDirectory, char *cacheDirectory);
uintptr_t webkit_website_data_manager_new(char *name1, char *value1, char *name2, char *value2, uintptr_t end);
uintptr_t webkit_web_context_new_with_website_data_manager(uintptr_t manager);
uintptr_t webkit_web_view_new_with_context(uintptr_t context);
uintptr_t webkit_settings_new(void);
void webkit_settings_set_hardware_acceleration_policy(uintptr_t settings, int32_t policy);
uintptr_t webkit_uri_scheme_request_get_http_method(uintptr_t request);
uintptr_t webkit_uri_scheme_request_get_http_body(uintptr_t request);
uintptr_t webkit_uri_scheme_response_new(uintptr_t stream, int64_t length);
void webkit_uri_scheme_response_set_status(uintptr_t response, uint32_t status, char *reason);
void webkit_uri_scheme_response_set_content_type(uintptr_t response, char *contentType);
void webkit_uri_scheme_response_set_http_headers(uintptr_t response, uintptr_t headers);
void webkit_uri_scheme_request_finish_with_response(uintptr_t request, uintptr_t response);
