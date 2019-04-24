## Goal
Users should be able to independently create, build as go plugins and load filters at runtime.
Adding and disabling plugins should be possible using @BotKube commands like-
- Adding filter:  `@BotKube filters add <filter-name> <URL>`
- List filters:   `@BotKube filters list`
- Delete filter:  `@BotKube filters delete <filter-name>`
- Disable filter: `@BotKube filters disable <filter-name>`
- Enable filter:  `@BotKube filters disable <filter-name>`

## Design
Downloading filter plugins: We will be using `wget` to download go plugin filters and store it into a dir (say /filters). Our filterengine will take care of reading, registering and executing filters. We can always reject invalid go plugins.
Enable/Disable plugins: we can use map to store state

## Limitations
- Since we are planning to store filters on the local filesystem, the downloaded filters can be lost if the BotKube pod restarts.
## Open questions:
- Not sure if using volumes to store downloaded plugins sounds is a good idea
- We can have a dedicated GitHub repo (like botkube-filters) which will contain standard plugins. And we can include these plugins as defaults in BotKube docker image.
