run:
	go run main.go

run_dummy_servers:
	go run dummy_server/test_server.go


load_test:
	npx autocannon -c 1000 -d 30 -m POST -b "name=Mouloud" "http://localhost:8080/hello?surname=Hasrane"

view_distribution:
	 wc -l dump/main_300{0,1,2}.log
