#define DOUBLE_TO_CDTIME_T_STATIC(d) ((cdtime_t)((d)*1073741824.0))
#define DOUBLE_TO_CDTIME_T(d)                                                  \
  (cdtime_t) { DOUBLE_TO_CDTIME_T_STATIC(d) }

cdtime_t cf_get_default_interval(void);
int cf_read(const char *filename);