node=10
delay=15 # second

start-sw:
	docker -H 10.200.8.149:4000 run -d --net dmp-overlay soulski/dmp \
		--cidr 173.17.0.0/29 \
		--net-if eth0 \
		--net wan

deploy-l:
	docker-compose up -d
	docker-compose scale node1=15
	@count=1; while [ $$count -le 50 ] ; do \
		sleep $(delay) ; \
		docker run -dit --net dmp_all dmp -c 173.17.0.2 --net-if eth0 --net wan \
		count=`expr $$count + 1`; \
	done
	echo "Finish start cluster" ;

deploy: build cluster-up
	docker-compose logs

cluster-up:
	docker-compose up -d
	docker-compose scale node2=$(node)

cluster-down:
	docker-compose down --rmi local

build:
	sh -c "'$(CURDIR)/script/build.sh'"

