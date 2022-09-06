//go:build integration

package e2e

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	appsv1cli "k8s.io/client-go/kubernetes/typed/apps/v1"
	deploymentutil "k8s.io/kubectl/pkg/util/deployment"
)

func setTestEnvsForDeploy(t *testing.T, appCfg Config, deployNsCli appsv1cli.DeploymentInterface, slackChannels map[string]*slack.Channel, discordChannels map[string]*discordgo.Channel) func(t *testing.T) {
	t.Helper()

	deployment, err := deployNsCli.Get(context.Background(), appCfg.Deployment.Name, metav1.GetOptions{})
	require.NoError(t, err)
	require.NotNil(t, deployment)

	containerIdx, exists := findContainerIdxByName(deployment, appCfg.Deployment.ContainerName)
	require.True(t, exists)

	envs := deployment.Spec.Template.Spec.Containers[containerIdx].Env

	originalEnvs := make([]v1.EnvVar, len(envs))
	copy(originalEnvs, envs)

	restoreDeployEnvsFn := func(t *testing.T) {
		t.Helper()
		t.Logf("Restoring envs for deployment...")
		deployment, err := deployNsCli.Get(context.Background(), appCfg.Deployment.Name, metav1.GetOptions{})
		require.NoError(t, err)
		require.NotNil(t, deployment)

		containerIdx, exists := findContainerIdxByName(deployment, appCfg.Deployment.ContainerName)
		require.True(t, exists)

		deployment.Spec.Template.Spec.Containers[containerIdx].Env = originalEnvs
		_, err = deployNsCli.Update(context.Background(), deployment, metav1.UpdateOptions{})
		require.NoError(t, err)
	}

	var newEnvs []v1.EnvVar

	if len(slackChannels) > 0 {
		slackEnabledEnvName := appCfg.Deployment.Envs.SlackEnabledName
		newEnvs = append(newEnvs, v1.EnvVar{Name: slackEnabledEnvName, Value: strconv.FormatBool(true)})

		for envName, slackChannel := range slackChannels {
			newEnvs = append(newEnvs, v1.EnvVar{Name: envName, Value: slackChannel.Name})
		}
	}

	if len(discordChannels) > 0 {
		discordEnabledEnvName := appCfg.Deployment.Envs.DiscordEnabledName
		newEnvs = append(newEnvs, v1.EnvVar{Name: discordEnabledEnvName, Value: strconv.FormatBool(true)})

		for envName, discordChannel := range discordChannels {
			newEnvs = append(newEnvs, v1.EnvVar{Name: envName, Value: discordChannel.ID})
		}
	}

	deployment.Spec.Template.Spec.Containers[containerIdx].Env = updateEnv(
		envs,
		newEnvs,
		nil,
	)

	_, err = deployNsCli.Update(context.Background(), deployment, metav1.UpdateOptions{})
	require.NoError(t, err)

	return restoreDeployEnvsFn
}

func waitForDeploymentReady(deployNsCli appsv1cli.DeploymentInterface, deploymentName string, waitTimeout time.Duration) error {
	var lastErr error
	err := wait.Poll(pollInterval, waitTimeout, func() (done bool, err error) {
		deployment, err := deployNsCli.Get(context.Background(), deploymentName, metav1.GetOptions{})
		if err != nil {
			lastErr = err
			return false, nil
		}

		condition := deploymentutil.GetDeploymentCondition(deployment.Status, appsv1.DeploymentAvailable)
		if condition == nil {
			lastErr = fmt.Errorf("deployment condition %q is nil", appsv1.DeploymentAvailable)
			return false, nil
		}

		return condition.Status == v1.ConditionTrue, nil
	})
	if err != nil {
		if err == wait.ErrWaitTimeout {
			return lastErr
		}
		return err
	}

	return nil
}

func findContainerIdxByName(deployment *appsv1.Deployment, containerName string) (int, bool) {
	notFoundResultFn := func() (int, bool) {
		return -1, false
	}

	if deployment == nil {
		return notFoundResultFn()
	}

	containers := deployment.Spec.Template.Spec.Containers
	for i, container := range containers {
		if container.Name != containerName {
			continue
		}

		return i, true
	}

	return notFoundResultFn()
}

// Original source: https://github.com/kubernetes/kubectl/blob/release-1.22/pkg/cmd/set/helper.go#L125-L157
// Copyright 2016 The Kubernetes Authors. Licensed under the Apache License, Version 2.0.

func findEnv(env []v1.EnvVar, name string) (v1.EnvVar, bool) {
	for _, e := range env {
		if e.Name == name {
			return e, true
		}
	}
	return v1.EnvVar{}, false
}

// updateEnv adds and deletes specified environment variables from existing environment variables.
// An added variable replaces all existing variables with the same name.
// Removing a variable removes all existing variables with the same name.
// If the existing list contains duplicates that are unrelated to the variables being added and removed,
// those duplicates are left intact in the result.
// If a variable is both added and removed, the removal takes precedence.
func updateEnv(existing []v1.EnvVar, env []v1.EnvVar, remove []string) []v1.EnvVar {
	out := []v1.EnvVar{}
	covered := sets.NewString(remove...)
	for _, e := range existing {
		if covered.Has(e.Name) {
			continue
		}
		newer, ok := findEnv(env, e.Name)
		if ok {
			covered.Insert(e.Name)
			out = append(out, newer)
			continue
		}
		out = append(out, e)
	}
	for _, e := range env {
		if covered.Has(e.Name) {
			continue
		}
		covered.Insert(e.Name)
		out = append(out, e)
	}
	return out
}
