#include "collectd.h"

#include "common.h"
#include "plugin.h"

#include <curl/curl.h>

struct nginx_s {
  char *name;
  char *host;
  char *url;
  char *user;
  char *pass;
  _Bool verify_peer;
  _Bool verify_host;
  char *cacert;
  char *ssl_ciphers;
  char *nginx_buffer;
  char nginx_curl_error[CURL_ERROR_SIZE];
  size_t nginx_buffer_size;
  size_t nginx_buffer_fill;
  int timeout;
  CURL *curl;
}; /* nginx_s */

typedef struct nginx_s nginx_t;

/* TODO: Remove this prototype */
static int nginx_read_host(user_data_t *user_data);

static void nginx_free(void *arg) {
  nginx_t *st = arg;

  if (st == NULL)
    return;

  sfree(st->name);
  sfree(st->host);
  sfree(st->url);
  sfree(st->user);
  sfree(st->pass);
  sfree(st->cacert);
  sfree(st->ssl_ciphers);
  sfree(st->nginx_buffer);
  if (st->curl) {
    curl_easy_cleanup(st->curl);
    st->curl = NULL;
  }
  sfree(st);
} /* nginx_free */

static size_t nginx_curl_callback(void *buf, size_t size, size_t nmemb,
                                   void *user_data) {
  size_t len = size * nmemb;
  nginx_t *st;

  st = user_data;
  if (st == NULL) {
    ERROR("nginx plugin: nginx_curl_callback: "
          "user_data pointer is NULL.");
    return (1);
  }

  if (len == 0)
    return (len);

  if ((st->nginx_buffer_fill + len) >= st->nginx_buffer_size) {
    char *temp;

    temp = realloc(st->nginx_buffer, st->nginx_buffer_fill + len + 1);
    if (temp == NULL) {
      ERROR("nginx plugin: realloc failed.");
      return (0);
    }
    st->nginx_buffer = temp;
    st->nginx_buffer_size = st->nginx_buffer_fill + len + 1;
  }

  memcpy(st->nginx_buffer + st->nginx_buffer_fill, (char *)buf, len);
  st->nginx_buffer_fill += len;
  st->nginx_buffer[st->nginx_buffer_fill] = 0;

  return (len);
} /* int nginx_curl_callback */

/* Configuration handling functiions
 * <Plugin nginx>
 *   <Instance "instance_name">
 *     URL ...
 *   </Instance>
 *   URL ...
 * </Plugin>
 */
static int config_add(oconfig_item_t *ci) {
  nginx_t *st;
  int status;

  st = calloc(1, sizeof(*st));
  if (st == NULL) {
    ERROR("nginx plugin: calloc failed.");
    return (-1);
  }

  st->timeout = -1;

  status = cf_util_get_string(ci, &st->name);
  if (status != 0) {
    sfree(st);
    return (status);
  }
  assert(st->name != NULL);

  for (int i = 0; i < ci->children_num; i++) {
    oconfig_item_t *child = ci->children + i;

    if (strcasecmp("URL", child->key) == 0)
      status = cf_util_get_string(child, &st->url);
    else if (strcasecmp("Host", child->key) == 0)
      status = cf_util_get_string(child, &st->host);
    else if (strcasecmp("User", child->key) == 0)
      status = cf_util_get_string(child, &st->user);
    else if (strcasecmp("Password", child->key) == 0)
      status = cf_util_get_string(child, &st->pass);
    else if (strcasecmp("VerifyPeer", child->key) == 0)
      status = cf_util_get_boolean(child, &st->verify_peer);
    else if (strcasecmp("VerifyHost", child->key) == 0)
      status = cf_util_get_boolean(child, &st->verify_host);
    else if (strcasecmp("CACert", child->key) == 0)
      status = cf_util_get_string(child, &st->cacert);
    else if (strcasecmp("SSLCiphers", child->key) == 0)
      status = cf_util_get_string(child, &st->ssl_ciphers);
    else if (strcasecmp("Timeout", child->key) == 0)
      status = cf_util_get_int(child, &st->timeout);
    else {
      WARNING("nginx plugin: Option `%s' not allowed here.", child->key);
      status = -1;
    }

    if (status != 0)
      break;
  }

  /* Check if struct is complete.. */
  if ((status == 0) && (st->url == NULL)) {
    ERROR("nginx plugin: Instance `%s': "
          "No URL has been configured.",
          st->name);
    status = -1;
  }

  if (status == 0) {
    char callback_name[3 * DATA_MAX_NAME_LEN];

    ssnprintf(callback_name, sizeof(callback_name), "nginx/%s/%s",
              (st->host != NULL) ? st->host : hostname_g,
              (st->name != NULL) ? st->name : "default");

    status = plugin_register_complex_read(
        /* group = */ NULL,
        /* name      = */ callback_name,
        /* callback  = */ nginx_read_host,
        /* interval  = */ 0, &(user_data_t){
                                 .data = st, .free_func = nginx_free,
                             });
  }

  if (status != 0) {
    nginx_free(st);
    return (-1);
  }

  return (0);
} /* int config_add */

static int config(oconfig_item_t *ci) {
  int status = 0;

  for (int i = 0; i < ci->children_num; i++) {
    oconfig_item_t *child = ci->children + i;

    if (strcasecmp("Instance", child->key) == 0)
      config_add(child);
    else
      WARNING("nginx plugin: The configuration option "
              "\"%s\" is not allowed here. Did you "
              "forget to add an <Instance /> block "
              "around the configuration?",
              child->key);
  } /* for (ci->children) */

  return (status);
} /* int config */

/* initialize curl for each host */
static int init_host(nginx_t *st) /* {{{ */
{
  assert(st->url != NULL);
  /* (Assured by `config_add') */

  if (st->curl != NULL) {
    curl_easy_cleanup(st->curl);
    st->curl = NULL;
  }

  if ((st->curl = curl_easy_init()) == NULL) {
    ERROR("nginx plugin: init_host: `curl_easy_init' failed.");
    return (-1);
  }

  curl_easy_setopt(st->curl, CURLOPT_NOSIGNAL, 1L);
  curl_easy_setopt(st->curl, CURLOPT_WRITEFUNCTION, nginx_curl_callback);
  curl_easy_setopt(st->curl, CURLOPT_WRITEDATA, st);

  curl_easy_setopt(st->curl, CURLOPT_USERAGENT, COLLECTD_USERAGENT);
  curl_easy_setopt(st->curl, CURLOPT_ERRORBUFFER, st->nginx_curl_error);

  if (st->user != NULL) {
#ifdef HAVE_CURLOPT_USERNAME
    curl_easy_setopt(st->curl, CURLOPT_USERNAME, st->user);
    curl_easy_setopt(st->curl, CURLOPT_PASSWORD,
                     (st->pass == NULL) ? "" : st->pass);
#else
    static char credentials[1024];
    int status;

    status = ssnprintf(credentials, sizeof(credentials), "%s:%s", st->user,
                       (st->pass == NULL) ? "" : st->pass);
    if ((status < 0) || ((size_t)status >= sizeof(credentials))) {
      ERROR("nginx plugin: init_host: Returning an error "
            "because the credentials have been "
            "truncated.");
      curl_easy_cleanup(st->curl);
      st->curl = NULL;
      return (-1);
    }

    curl_easy_setopt(st->curl, CURLOPT_USERPWD, credentials);
#endif
  }

  curl_easy_setopt(st->curl, CURLOPT_URL, st->url);
  curl_easy_setopt(st->curl, CURLOPT_FOLLOWLOCATION, 1L);
  curl_easy_setopt(st->curl, CURLOPT_MAXREDIRS, 50L);

  curl_easy_setopt(st->curl, CURLOPT_SSL_VERIFYPEER, (long)st->verify_peer);
  curl_easy_setopt(st->curl, CURLOPT_SSL_VERIFYHOST, st->verify_host ? 2L : 0L);
  if (st->cacert != NULL)
    curl_easy_setopt(st->curl, CURLOPT_CAINFO, st->cacert);
  if (st->ssl_ciphers != NULL)
    curl_easy_setopt(st->curl, CURLOPT_SSL_CIPHER_LIST, st->ssl_ciphers);

#ifdef HAVE_CURLOPT_TIMEOUT_MS
  if (st->timeout >= 0)
    curl_easy_setopt(st->curl, CURLOPT_TIMEOUT_MS, (long)st->timeout);
  else
    curl_easy_setopt(st->curl, CURLOPT_TIMEOUT_MS,
                     (long)CDTIME_T_TO_MS(plugin_get_interval()));
#endif

  return (0);
} /* }}} int init_host */

static void submit_value(const char *type, const char *type_instance,
                         value_t value, nginx_t *st) {
  value_list_t vl = VALUE_LIST_INIT;

  vl.values = &value;
  vl.values_len = 1;

  if (st->host != NULL)
    sstrncpy(vl.host, st->host, sizeof(vl.host));

  sstrncpy(vl.plugin, "nginx", sizeof(vl.plugin));
  if (st->name != NULL)
    sstrncpy(vl.plugin_instance, st->name, sizeof(vl.plugin_instance));

  sstrncpy(vl.type, type, sizeof(vl.type));
  if (type_instance != NULL)
    sstrncpy(vl.type_instance, type_instance, sizeof(vl.type_instance));

  plugin_dispatch_values(&vl);
} /* void submit_value */

static void submit_derive(const char *type, const char *type_instance,
                          derive_t d, nginx_t *st) {
  submit_value(type, type_instance, (value_t){.derive = d}, st);
} /* void submit_derive */

static void submit_gauge(const char *type, const char *type_instance, gauge_t g,
                         nginx_t *st) {
  submit_value(type, type_instance, (value_t){.gauge = g}, st);
} /* void submit_gauge */

static int nginx_read_host(user_data_t *user_data) /* {{{ */
{
  char *ptr;
  char *lines[16];
  int lines_num = 0;
  char *saveptr;

  char *fields[16];
  int fields_num;

  int status;

  nginx_t *st;
  st = user_data->data;

  if (st->curl == NULL) {
    status = init_host(st);
    if (status != 0)
      return (-1);
  }
  assert(st->curl != NULL);

  if (st->url == NULL)
    return (-1);

  st->nginx_buffer_fill = 0;
  if (curl_easy_perform(st->curl) != CURLE_OK) {
    WARNING("nginx plugin: curl_easy_perform failed: %s", st->nginx_curl_error);
    return (-1);
  }

  ptr = st->nginx_buffer;
  saveptr = NULL;
  while ((lines[lines_num] = strtok_r(ptr, "\n\r", &saveptr)) != NULL) {
    ptr = NULL;
    lines_num++;

    if (lines_num >= 16)
      break;
  }

  for (int i = 0; i < lines_num; i++) {
    fields_num =
        strsplit(lines[i], fields, (sizeof(fields) / sizeof(fields[0])));

    if (fields_num == 3) {
      if ((strcmp(fields[0], "Active") == 0) &&
          (strcmp(fields[1], "connections:") == 0)) {
        submit_gauge("nginx_connections", "active", atoll(fields[2]), st);
      } else if ((atoll(fields[0]) != 0) && (atoll(fields[1]) != 0) &&
                 (atoll(fields[2]) != 0)) {
        submit_derive("connections", "accepted", atoll(fields[0]), st);
        /* TODO: The legacy metric "handled", which is the sum of "accepted" and
         * "failed", is reported for backwards compatibility only. Remove in the
         * next major version. */
        submit_derive("connections", "handled", atoll(fields[1]), st);
        submit_derive("connections", "failed", (atoll(fields[0]) - atoll(fields[1])), st);
        submit_derive("nginx_requests", NULL, atoll(fields[2]), st);
      }
    } else if (fields_num == 6) {
      if ((strcmp(fields[0], "Reading:") == 0) &&
          (strcmp(fields[2], "Writing:") == 0) &&
          (strcmp(fields[4], "Waiting:") == 0)) {
        submit_gauge("nginx_connections", "reading", atoll(fields[1]), st);
        submit_gauge("nginx_connections", "writing", atoll(fields[3]), st);
        submit_gauge("nginx_connections", "waiting", atoll(fields[5]), st);
      }
    }
  }

  st->nginx_buffer_fill = 0;

  return (0);} /* }}} int nginx_read_host */

static int nginx_init(void) /* {{{ */
{
  /* Call this while collectd is still single-threaded to avoid
   * initialization issues in libgcrypt. */
  curl_global_init(CURL_GLOBAL_SSL);
  return (0);
} /* }}} int nginx_init */

void module_register(void) {
  plugin_register_complex_config("nginx", config);
  plugin_register_init("nginx", nginx_init);
} /* void module_register */
