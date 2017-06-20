#include <semaphore.h>
#include <stdio.h>
#include <signal.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>

#include "collectd.h"
#include "configfile.h"
#include "plugin.h"

static const char *conf_file = "/etc/collectd/collectd.conf";

sem_t reload_sem;

void handle_hup(int sig) {
	sem_post(&reload_sem);
}

void start() {
	plugin_init_ctx();
	cf_read(conf_file);

	init_collectd();
	interval_g = cf_get_default_interval();

	plugin_init_all();

	plugin_read_all();
}

void reload() {
	printf("reload collectd plugins requested\n");

	plugin_shutdown_for_reload();
	plugin_init_ctx();
	cf_read(conf_file);
	plugin_init_for_reload();
}


void main(int argc, char *argv[]) {
	// Handle the metadata plugin trying to call this proc with the -h flag to
	// get version.  If we don't do this the process spawns recursively until
	// the kernel stops it.
	if (argc > 1) {
		printf("Usage: neomock");
		printf("collectd version: 5.7.0\n");
		exit(0);
	}

	start();

	// Init to zero (locked) and let the signal handler unlock it
	sem_init(&reload_sem, 0, 0);

	if (signal(SIGHUP, handle_hup) == SIG_ERR) {
		printf("Error attaching reload signal handler");
		exit(1);
	}

	while (1) {
		sem_wait(&reload_sem);
		reload();
		// We could get back-to-back reloads here if multiple HUPs were sent in
		// quick succession but they will always be done in serial order.
	}
}

