package k8s

import (
	"fmt"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/retry"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Same as WaitUntilPodsAvailableE, but terminates the test instead of returning an error.
func WaitUntilPodsAvailable(
	t *testing.T,
	options *k8s.KubectlOptions,
	filters v1.ListOptions,
	minPods int,
	sleepDuration time.Duration,
	sleepBetweenRetries time.Duration,
) {
	require.NoError(t, WaitUntilPodsAvailableE(t, options, filters, minPods, sleepDuration, sleepBetweenRetries))
}

// WaitUntilPodsAvailableE waits until pods satisfying the provided filters are all available.
// Availability is defined via terrastruct's k8s.IsPodAvailable function, which requires the Pod
// to be Running and all of its containers to be started and ready.
// Technically this condition is satisfied if no Pods exists, the minPods argument helps with this edge case.
func WaitUntilPodsAvailableE(
	t *testing.T,
	options *k8s.KubectlOptions,
	filters v1.ListOptions,
	minPods int,
	sleepDuration time.Duration,
	sleepBetweenRetries time.Duration,
) error {
	retries := int(sleepDuration / sleepBetweenRetries)
	message, err := retry.DoWithRetryE(
		t,
		fmt.Sprintf("WaitUntilPodsAvailable(%s)", formatSelectors(filters)),
		retries,
		sleepBetweenRetries,
		func() (string, error) {
			pods, err := k8s.ListPodsE(t, options, filters)
			if err != nil {
				return "", err
			}
			if len(pods) < minPods {
				return "", fmt.Errorf(
					"found %d pods (%s), need at least %d",
					len(pods), formatSelectors(filters), minPods,
				)
			}
			for _, pod := range pods {
				if !k8s.IsPodAvailable(&pod) {
					return "", k8s.NewPodNotAvailableError(&pod)
				}
			}
			return fmt.Sprintf("Pods (%s) are now available.", formatSelectors(filters)), nil
		},
	)
	if err != nil {
		t.Logf(
			"Timed out waiting for pods (%s) to be available: %s",
			formatSelectors(filters), err,
		)
		return err
	}
	t.Log(message)
	return nil
}

func formatSelectors(listOptions v1.ListOptions) string {
	return fmt.Sprintf("LabelSelector: %q, FieldSelector: %q",
		listOptions.LabelSelector, listOptions.FieldSelector,
	)
}
