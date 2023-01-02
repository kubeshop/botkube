# RBAC Proof of concept

This PoC is a part of the [RBAC proposal](../../proposal/2022-12-23-rbac.md). It shows the way how we can use user-provided Roles and ClusterRoles to execute `kubectl` commands. While this PoC bases on built-in `kubectl` executor, the proposal itself describes also the API for plugins. The implementation will be very similar, but we need to make such modifications in the Default Executors instead of modifying the `kubectl` executor.

## Usage

1. Create `rbac-test` and `rbac-test2` channels in your Slack workspace you'll use for testing.
1. Set up a Kubernetes cluster, e.g. with `k3d` or `colima`.
1. Invite Botkube bot to these channels.
1. Clone this repository and check out the PoC pull request with GH CLI:

   ```bash
   gh pr checkout 909
   ```

1. Deploy Botkube with the following values using local chart from the PR:

   ```yaml
   image:
     repository: kubeshop/pr/botkube
     tag: 909-PR
   communications:
     "default-group":
       socketSlack:
         enabled: true
         appToken: "xapp-..." # TODO: Set app token
         botToken: "xoxb-..." # TODO: Set bot token
         notification:
           type: "short"
         channels:
           "default":
             name: rbac-test
             bindings:
               executors:
                 - kubectl-read-only
               sources:
                 - k8s-err-events
           "second":
             name: rbac-test2
             bindings:
               executors:
                 - kubectl-read-only
               sources:
                 - k8s-err-events
   executors:
     kubectl-read-only:
       kubectl:
         enabled: true
   settings:
     clusterName: dev

   analytics:
     disable: true
   ```

   ```bash
   helm install botkube --namespace botkube ./helm/botkube --wait --create-namespace -f ~/rbac-poc-values.yaml
   ```

1. Apply Role and ClusterRole resources:

   ```bash
   kubectl apply -f ./docs/investigation/rbac/assets
   ```

1. Test getting pods in the `rbac-test` channel:

   ```
   @Botkube get po
   ```

   ```
   @Botkube get po -n botkube
   ```

   Do the same in the `rbac-test2` channel. You should get an error:

   ```
   Error from server (Forbidden): services is forbidden: User "{your-email}" cannot list resource "pods" in API group "" in the namespace "default"
   exit status 1
   ```

1. Test getting services on the first channel:

   This will work:

   ```
   @Botkube get svc
   ```

   But this command won't:

   ```
   @Botkube get svc -n botkube
   ```

   Same with other resources:

   ```
   @Botkube get ingress
   ```

1. In the second channel (`rbac-test2`), get all deployments:

   ```
   @Botkube get deploy -A
   ```

   This command won't work in the first channel.

You can change permissions for Roles and ClusterRoles in runtime and they will be taken into account on the next command execution.

## Process isolation

We have a full control over how a given plugin is run. That means we can use e.g. `chroot` to isolate the plugin process from the rest of the system, in order to avoid reading sensitive credentials or other Kubeconfigs with different permissions.

I tested it successfully with a simple separate Go app on my machine. However, when I modified the code in Botkube codebase, the plugin's gRPC server exits with an error. 

```go
 	for key, path := range bins {
 		pluginLogger, stdoutLogger, stderrLogger := NewPluginLoggers(logger, key, pluginType)
 
 		dir, file := filepath.Split(path)
 		//nolint:gosec // warns us about 'Subprocess launching with variable', but we are the one that created that variable.
 		cmd := exec.Command("./" + file)
 		cmd.Dir = "/"
 		cmd.SysProcAttr = &syscall.SysProcAttr{
 			Chroot:     dir,
 		}
 
 		cli := plugin.NewClient(&plugin.ClientConfig{
 			Plugins: pluginMap,
 			Cmd:              newPluginOSRunCommand(path),
 			Plugins:          pluginMap,
 			Cmd:              cmd,
 			AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
 			HandshakeConfig: plugin.HandshakeConfig{
      // ...
		})
    
    // ...
  }
```

I didn't debug the issue, but I suspect some additional symlinks might be needed to make it work, as well as modifying the Docker image. We decided to come back to this topic during actual implementation.

## References

See the pull request [#909](https://github.com/kubeshop/botkube/pull/909) for source code changes needed for the basic RBAC PoC.
