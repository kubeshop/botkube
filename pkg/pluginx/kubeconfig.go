package pluginx

import "context"

func PersistKubeConfig(ctx context.Context, kc []byte) (string, func(context.Context) error, error)
