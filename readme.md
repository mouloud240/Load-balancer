## Overview
A Very simple implementation of a golang load l7 load balancer with a simple round-robin algorithm. This load balancer can distribute incoming HTTP requests to multiple backend servers, improving the performance and reliability of your web applications.


## Disclaimer
This is just for educational purposes and should not be used in production environments.

This does not claim it is the best approach nor the fastest , just a simple one.

## Vision This is really just an idea I had in mind on how would I implement it very simply , without any external dependencies or advanced techniques, and you it is one of those ideas that you just want to code quickly and see if it does actually work , we don't get to do a lot of those anymore work and all + the ai raise that overcomplicates and bloats the whole thing.

With that being said I intend to keep building this into a prod ready lb , that is simple to use and plug to your system if you need a simple lb and reversy proxy behavior.

Starting with customizablity of servers and running params , so you can actually use this a cli tool (some .yaml file or .json file to configure it)
And dockerising this to imbrace the spirit of microservices and containerization.
## Running the Load Balancer
To run the load balancer, follow these steps:
1. Clone the repository to your local machine.
`git clone github.com/mouloud240/load-balancer`
2. Navigate to the project directory.
`cd load-balancer`
3. Run the make file command
`make run`
4. The repo also comes with 3 dummy servers that you can launch to test
`make run_dummy_servers`


You can also run a simple auto cannon load test using :
`make load_test`

And view the end distribution of requests across the servers using:
`make view_distribution`

