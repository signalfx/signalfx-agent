#include <stdio.h>

#include "collectd.h"
#include "configfile.h"
#include "plugin.h"

cdtime_t interval_g;

void init_collectd() { printf("stub init collectd\n"); }
void plugin_init_ctx() {}
int plugin_init_all() { return 0; }
void plugin_read_all() {}
int plugin_shutdown_all() { return 0; }
void plugin_shutdown_for_reload() {}
int plugin_init_for_reload() { return 0; }
cdtime_t cf_get_default_interval() { return DOUBLE_TO_CDTIME_T(10.0); }
int cf_read(const char *filename) { return 0; }