//go:build integration

package e2e

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	appsv1cli "k8s.io/client-go/kubernetes/typed/apps/v1"
	deploymentutil "k8s.io/kubectl/pkg/util/deployment"
)

func setTestEnvsForDeploy(t *testing.T, appCfg Config, deployNsCli appsv1cli.DeploymentInterface, channelName string) func(t *testing.T) {
	t.Helper()

	slackEnabledEnvName := appCfg.Deployment.Envs.SlackEnabledName
	slackChannelIDEnvName := appCfg.Deployment.Envs.SlackChannelIDName

	deployment, err := deployNsCli.Get(context.Background(), appCfg.Deployment.Name, metav1.GetOptions{})
	require.NoError(t, err)
	require.NotNil(t, deployment)

	containerIdx, exists := findContainerIdxByName(deployment, appCfg.Deployment.ContainerName)
	require.True(t, exists)

	envs := deployment.Spec.Template.Spec.Containers[containerIdx].Env

	var originalEnvs []v1.EnvVar
	channelIDEnv, exists := findEnv(envs, slackChannelIDEnvName)
	if exists {
		originalEnvs = append(originalEnvs, channelIDEnv)
	}
	enabledEnv, exists := findEnv(envs, slackEnabledEnvName)
	if exists {
		originalEnvs = append(originalEnvs, enabledEnv)
	}

	restoreDeployEnvsFn := func(t *testing.T) {
		t.Helper()
		t.Logf("Restoring envs for deployment...")
		deployment, err := deployNsCli.Get(context.Background(), appCfg.Deployment.Name, metav1.GetOptions{})
		require.NoError(t, err)
		require.NotNil(t, deployment)

		containerIdx, exists := findContainerIdxByName(deployment, appCfg.Deployment.ContainerName)
		require.True(t, exists)

		deployment.Spec.Template.Spec.Containers[containerIdx].Env = updateEnv(envs,
			nil,
			[]string{slackEnabledEnvName, slackChannelIDEnvName},
		)

		if len(originalEnvs) > 0 {
			deployment.Spec.Template.Spec.Containers[containerIdx].Env = updateEnv(envs, originalEnvs, nil)
		}
		_, err = deployNsCli.Update(context.Background(), deployment, metav1.UpdateOptions{})
		require.NoError(t, err)
	}

	deployment.Spec.Template.Spec.Containers[containerIdx].Env = updateEnv(
		envs,
		[]v1.EnvVar{
			{Name: slackEnabledEnvName, Value: strconv.FormatBool(true)},
			{Name: slackChannelIDEnvName, Value: channelName},
		},
		nil,
	)

	_, err = deployNsCli.Update(context.Background(), deployment, metav1.UpdateOptions{})
	require.NoError(t, err)

	return restoreDeployEnvsFn
}

func waitForDeploymentReady(deployNsCli appsv1cli.DeploymentInterface, deploymentName string, waitTimeout time.Duration) error {
	// Wait for deployment ready
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
	containerIdx := -1

	if deployment == nil {
		return containerIdx, false
	}

	containers := deployment.Spec.Template.Spec.Containers
	for i, container := range containers {
		if container.Name != containerName {
			continue
		}

		containerIdx = i
	}

	return containerIdx, containerIdx != -1
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
