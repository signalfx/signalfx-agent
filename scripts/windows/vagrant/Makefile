WIN_VER ?= server_2008

ifneq ($(OS), Windows_NT)
VAGRANT_NOT_CREATED := $(shell cd $(WIN_VER); vagrant status | grep "not created")

.PHONY: win-vagrant-base-box
win-vagrant-base-box:
	WIN_VER=$(WIN_VER) ./build_vagrant_base_box.sh

.PHONY: win-vagrant-reload
win-vagrant-reload:
	cd $(WIN_VER); vagrant reload

.PHONY: win-vagrant-up
win-vagrant-up:
	cd $(WIN_VER); vagrant up
	# only reload if the vagrant wasn't created
ifneq ($(strip $(VAGRANT_NOT_CREATED)),)
	cd $(WIN_VER); vagrant reload
endif


.PHONY: win-vagrant-suspend
win-vagrant-suspend:
	cd $(WIN_VER); vagrant suspend

.PHONY: win-vagrant-destroy
win-vagrant-destroy:
	cd $(WIN_VER); vagrant destroy

.PHONY: win-vagrant-provision
win-vagrant-provision:
	cd $(WIN_VER); vagrant provision
endif
