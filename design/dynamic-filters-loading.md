## Goal
Users should be able to independently create, build as go plugins and load filters at runtime.
Adding and disabling plugins should be possible using @BotKube commands like-
- Adding filter:  `@BotKube filters add <filter-name> <URL>`
- List filters:   `@BotKube filters list`
- Delete filter:  `@BotKube filters delete <filter-name>`
- Disable filter: `@BotKube filters disable <filter-name>`
- Enable filter:  `@BotKube filters disable <filter-name>`

## Design
### Building go plugins:
The only change the existing filter go code structure (https://www.botkube.io/filters/#a-writing-a-filter) need is having `main` package instead of `filters`. A go plugin can be built from a go file using `go build -buildmode=plugin -o dest.so src.go`. This will create a `.so` go plugin
### Downloading filter plugins:
We will be using `wget` to download go plugin filters and store it into a dir (say /filters). 
### Reading plugins:
Our filterengine will take care of reading, registering and executing filters. We can always reject invalid go plugins.
   - Filterengine will basically get all the `.so` files in `/filters` dir.
   - Open filter with `plugin.Open(f)` and verify if filter implements `Filter` interface using `plug.Lookup("Filter")`
   - If yes, we register the plugin to DefaultFilterEngine
    
### Enable/Disable plugins:
we can use map to store filter state

## Limitations
- Since we are planning to store filters on the local filesystem, the downloaded filters can be lost if the BotKube pod restarts.

## Open questions:
- Not sure if using volumes to store downloaded plugins sounds is a good idea
- We can have a dedicated GitHub repo (like botkube-filters) which will contain standard plugins. And we can include these plugins as defaults in BotKube docker image.

## References
- https://golang.org/pkg/plugin/
- https://medium.com/learning-the-go-programming-language/writing-modular-go-programs-with-plugins-ec46381ee1a9 
