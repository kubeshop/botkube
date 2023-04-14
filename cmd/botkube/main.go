package main

import "fmt"

func main() {
	fmt.Println(`level=info msg="Starting integration                                  integration=prometheus"`)
	fmt.Println(`level=info msg="Starting integration                                  integration=loki"`)
	fmt.Println(`level=info msg="Starting integration                                  integration=argocd"`)
	fmt.Println(`level=info msg="Starting integration                                  integration=segment"`)
	fmt.Println(`level=info msg="Starting server on address ":2113"            component="Lifecycle server"`)
	fmt.Println(`level=info msg="Analytics disabled via configuration settings."`)
	fmt.Println(`level=info msg="Registering filter "ObjectAnnotationChecker" (enabled: true)...  component="Filter Engine"`)
	fmt.Println(`level=info msg="Registering filter "NodeEventsChecker" (enabled: true)...  component="Filter Engine"`)
	fmt.Println(`level=info msg="Starting Plugin Manager for all enabled plugins" component="Plugin Manager`)
	fmt.Println(`level=info msg="Starting server on address ":2112"            component="Metrics server"`)
	fmt.Println(`level=error msg="While executing request: dial tcp 192.168.65.2:3000: connect: connection refused. component="Metrics server"`)
	fmt.Println(`level=info msg="Shutdown requested. Sending final message...  component=Controller"`)
	fmt.Println(`level=info msg="Shutdown requested. Finishing...              integration=prometheus"`)
	fmt.Println(`level=info msg="Shutdown requested. Finishing...              component="Metrics server"`)
	fmt.Println(`level=info msg="Shutdown requested. Finishing...              integration=loki"`)
	fmt.Println(`level=info msg="Shutdown requested. Finishing...              component="Lifecycle server"`)
}
