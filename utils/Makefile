#=============================================================================
#     FileName: Makefile
#         Desc: utils's Makefile 
#       Author: ato.ye
#        Email: ato.ye@ucloud.cn
#     HomePage: http://www.ucloud.cn
#      Version: 0.0.1
#   LastChange: 2016-04-20 18:18:19
#      History:
#=============================================================================

UFRAMEWORK_PATH=..

SRCDIRS_EXCLUDE = new_project zk_admin update_to_v0.2
SRCDIRS_ALL = $(sort $(subst /,,$(dir $(wildcard */*))))
SRCDIRS = $(filter-out $(SRCDIRS_EXCLUDE), $(SRCDIRS_ALL))

PKGDIRS_EXCLUDE = $(GOROOT)/pkg
PKGDIRS_ALL = $(addsuffix /pkg, $(subst :, ,$(GOPATH)))
PKGDIRS = $(filter-out $(PKGDIRS_EXCLUDE), $(PKGDIRS_ALL))


all:zkadmin
	@for subdir in $(SRCDIRS);do \
		cd $$subdir; go install; cd $(CURDIR); \
	done 

zkadmin:
	@cd zk_admin; \
	CGO_ENABLED=0 go build zk_admin.go;\
	cd $(CURDIR);

show:
	@echo "==================src====================="
	@echo SRCDIRS_ALL: $(SRCDIRS_ALL)
	@echo SRCDIRS_EXCLUDE: $(SRCDIRS_EXCLUDE)
	@echo SRCDIRS: $(SRCDIRS)
	@echo "==================pkg====================="
	@echo PKGDIRS_EXCLUDE: $(PKGDIRS_EXCLUDE)
	@echo PKGDIRS_ALL: $(PKGDIRS_ALL)
	@echo PKGDIRS: $(PKGDIRS)
	@echo "================clean======================"
	@for subdir in $(PKGDIRS); do \
		cd $$subdir;\
		result=`find . |grep uframework|head -n1|awk -F"." '{print $$2}'`; \
		if [ -n "$$result" ];then \
			echo clean_PKGDIRS:$$subdir$$result; \
		fi \
	done

clean: 
	@cd zk_admin;\
	rm -f zk_admin;\
	cd $(CURDIR)
	@for subdir in $(PKGDIRS); do \
		cd $$subdir;\
		result=`find . |grep utils |head -n1|awk -F"." '{print $$2}'`; \
		if [ -n "$$result" ];then \
			cd $$subdir$$result; rm -rf *; \
		fi \
	done
