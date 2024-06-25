package kuber

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"gitea.hama.de/LFS/go-logger"
	"gitea.hama.de/LFS/lfsx-web/controller/internal/models"
	"gitea.hama.de/LFS/lfsx-web/controller/pkg/utils"
	batchv1 "k8s.io/api/batch/v1"
	modelsv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// template data for pod creation
type podCreateTemplateData struct {
	Username           string
	Db                 string
	LfsServiceEndpoint string
	LfsConfigDir       string
	Image              string
	BaseName           string
	Namespace          string
	IsPlaceholder      bool
	ImageVersion       string
}

// found returns an pod that is assigned for the given user.
// If no pod was found, a new pod will be created / assigned.
// In such a case 'true' will be returned as the second parameter
func (k *Kuber) GetPodByUser(user *models.User) (*modelsv1.Pod, bool, error) {

	// Get an already existing pod
	if p, err := k.findPodForUser(user); err != nil {
		return nil, false, err
	} else if p != nil {
		return p, false, nil
	}

	// Create a new pod for the user
	pod, err := k.createJobForUserAbstract(user)
	if err != nil {
		return pod, true, err
	}

	return pod, true, err
}

// findPodForUser returns an already started pod for the given user.
// If no pod was found nil will be returned
func (k *Kuber) findPodForUser(user *models.User) (*modelsv1.Pod, error) {

	// Filter condition for the container
	sel := metav1.LabelSelector{
		MatchLabels: map[string]string{
			"app":         utils.GetEnvString("BASE_APP_NAME", "lfsx-web") + "-lfs",
			"db":          strings.ToLower(user.Database.String()),
			"user":        strings.ToLower(user.DbUser),
			"placeholder": "false",
			// We don't filter after the imageVersion. The user would not be abled to go to his
			// old pod after an update of the controller / LFS.X.
			// If we want to force to user to reconnect, add a helm hook!
			//"imageVersion": k.appConfig.GetLfsImageVersion(),
		},
	}

	// Make the request
	pods, err := k.Client.CoreV1().Pods(k.Namespace).List(context.Background(), metav1.ListOptions{LabelSelector: metav1.FormatLabelSelector(&sel)})
	if err != nil {
		return nil, fmt.Errorf("failed to get available pods: %s", err)
	}

	// A pod does already exist
	if len(pods.Items) > 0 {
		logger.Debug("Found pod for user%q: %s", user.DbUser, pods.Items[0].Status.PodIP)
		return &pods.Items[0], nil
	}

	return nil, nil
}

// createJobForUserAbstract creates a new job for the given user or changes an existing
// placeholder job so that it can be used for this user.
// This function does hide the implemntation detail
func (k *Kuber) createJobForUserAbstract(user *models.User) (*modelsv1.Pod, error) {

	// Try to get a placeholder job that is not used already
	sel := metav1.LabelSelector{
		MatchLabels: map[string]string{
			"appGeneric":   "lfs",
			"placeholder":  "true",
			"imageVersion": k.appConfig.GetLfsImageVersion(),
		},
	}

	for {
		// Execute the request
		jobs, err := k.Client.BatchV1().Jobs(k.Namespace).List(context.Background(), metav1.ListOptions{
			LabelSelector: metav1.FormatLabelSelector(&sel),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to find placeholders: %s", err)
		}

		// Sort the array so that the oldest created jobs will be used first
		sort.Slice(jobs.Items, func(a, b int) bool {
			return jobs.Items[a].CreationTimestamp.Time.Before(jobs.Items[b].CreationTimestamp.Time)
		})

		// Get patch string as "MergePatchType"
		patch := struct {
			Metadata struct {
				Labels          map[string]string `json:"labels"`
				ResourceVersion string            `json:"resourceVersion"`
			} `json:"metadata"`
		}{}
		patch.Metadata.Labels = map[string]string{
			"db":          strings.ToLower(user.Database.String()),
			"user":        strings.ToLower(user.DbUser),
			"placeholder": "false",
		}

		// Try to update a job with the specified ressource version. If that does succeed the job wasn't used before
		for _, job := range jobs.Items {
			ressourceVersion := job.ResourceVersion
			patch.Metadata.ResourceVersion = ressourceVersion

			patchJson, err := json.Marshal(patch)
			if err != nil {
				return nil, fmt.Errorf("failed druing marshal of patch json: %s", err)
			}

			_, err = k.Client.BatchV1().Jobs(k.Namespace).Patch(
				context.Background(),
				job.Name,
				types.MergePatchType,
				patchJson,
				metav1.PatchOptions{},
			)

			if err == nil {
				// Get random identifier from username label
				identifier := job.Labels["user"]
				sel.MatchLabels["user"] = identifier
				logger.Debug("Found and updated placeholder job %q for user %q", identifier, user.Username)

				// Get the pod name that was created for the job
				pods, err := k.Client.CoreV1().Pods(k.Namespace).List(context.Background(), metav1.ListOptions{
					LabelSelector: metav1.FormatLabelSelector(&sel),
				})
				if err != nil {
					return nil, fmt.Errorf("failed to find pods: %s", err)
				}

				if len(pods.Items) != 1 {
					return nil, fmt.Errorf("found no pod for job identifier")
				}

				// Build patch data
				patch.Metadata.ResourceVersion = pods.Items[0].ResourceVersion
				patchJson, err := json.Marshal(patch)
				if err != nil {
					return nil, fmt.Errorf("failed druing marshal of patch json: %s", err)
				}

				// Execute request
				pod, err := k.Client.CoreV1().Pods(k.Namespace).Patch(
					context.Background(), pods.Items[0].Name, types.MergePatchType, patchJson, metav1.PatchOptions{},
				)

				// Create new pods again
				if err == nil {
					go func(numberOfPlaceholders int) {
						if _, err := k.CreatePlaceholderJob(); err != nil {
							logger.Warning("Failed to create placeholder job: %s", err)
						}

						// Create another placeholder if no more than two placeholder pods were available
						if numberOfPlaceholders < 2 {
							logger.Debug("Starting another placeholder because current count is %d", numberOfPlaceholders)
							if _, err := k.CreatePlaceholderJob(); err != nil {
								logger.Warning("Failed to create placeholder job: %s", err)
							}
						}
					}(len(jobs.Items))
				}

				return pod, err
			} else {
				logger.Debug("Failed to update job %q with ressource version %q. It may got updated by another request: %s", job.Name, ressourceVersion, err)
			}
		}

		// No pods were found
		if len(jobs.Items) == 0 {
			break
		}
	}

	// As a last option create an own pod specific for the user
	go func() {
		for i := 0; i < 2; i++ {
			if _, err := k.CreatePlaceholderJob(); err != nil {
				logger.Warning("Failed to create placeholder job: %s", err)
			}
		}
	}()

	return k.createJodForUser(user)
}

// GetPlaceholders returns a list of placeholder jobs that can be assigned
// to a specifc user
func (k *Kuber) GetPlaceholders() (*batchv1.JobList, error) {

	// Only select plceholders for the current version
	sel := metav1.LabelSelector{
		MatchLabels: map[string]string{
			"appGeneric":   "lfs",
			"placeholder":  "true",
			"imageVersion": k.appConfig.GetLfsImageVersion(),
		},
	}

	return k.Client.BatchV1().Jobs(k.Namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(&sel),
	})
}

// createJodForUser creates a new Job that runs a Pod with the LFS for the given
// user and starts it up.
// This method blocks until the container is up and running
func (k *Kuber) createJodForUser(user *models.User) (*modelsv1.Pod, error) {
	logger.Debug("Creating job for user %q", user.DbUser)

	// Parse template data
	lfsConfigDir := "/opt/lfs-user/config-dev"
	if k.appConfig.Production {
		lfsConfigDir = "/opt/lfs-user/config-prod"
	}
	obj, _, err := k.getRessourceFromFile("deployment-lfs.yaml",
		podCreateTemplateData{
			Username:           strings.ToLower(user.DbUser),
			Db:                 strings.ToLower(user.Database.String()),
			LfsServiceEndpoint: k.appConfig.LfsServiceEndpoint,
			LfsConfigDir:       lfsConfigDir,
			Image:              k.appConfig.GetLfsImage(),
			ImageVersion:       k.appConfig.GetLfsImageVersion(),
			BaseName:           utils.GetEnvString("BASE_APP_NAME", "lfsx-web"),
			Namespace:          k.Namespace,
		},
	)
	if err != nil {
		return nil, err
	}

	// Create pod ressource
	_, err = k.Client.BatchV1().Jobs(k.Namespace).Create(context.TODO(), obj.(*batchv1.Job), metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create job: %s", err)
	}

	// Wait until pod is up and running
	watchOptions := metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(&metav1.LabelSelector{
			MatchLabels: map[string]string{
				"db":   strings.ToLower(user.Database.String()),
				"user": strings.ToLower(user.DbUser),
			},
		}),
	}

	podwatch, err := k.Client.CoreV1().Pods(k.Namespace).Watch(context.TODO(), watchOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for created pod: %s", err)
	}

	// Use a timeout of 8 seconds for pod readiness
	logger.Trc("Waiting for pod to become ready")
	timeout := time.After(10 * time.Second)
	for {
		select {
		case pr := <-podwatch.ResultChan():
			p := pr.Object.(*modelsv1.Pod)

			// Check if the pod is running and is ready. For the readiness an array of
			// states is given -> loop until ready state was found with message "True"
			if p.Status.Phase == modelsv1.PodRunning {
				for _, a := range p.Status.Conditions {
					if a.Type == modelsv1.ContainersReady && a.Status == "True" {
						logger.Trc("Pod is redy now")
						return p, nil
					}
				}
			} else {
				logger.Trc("Pod was changed but it is not ready yet. Current phase: %s", p.Status.Phase)
			}
		case <-timeout:
			return nil, fmt.Errorf("timeout while waiting for pod readiness")
		}
	}
}

// CreatePlaceholderJob creates a new Job that runs a Pod with the LFS without logging in.
// This method blocks until the container got created
func (k *Kuber) CreatePlaceholderJob() (*batchv1.Job, error) {
	// Generate a random string to identify the pod with the job
	identifier, _ := utils.GenerateRandomString(24)
	identifier = "p" + identifier + "p"

	logger.Debug("Creating placeholder job with identifier %q", identifier)

	// Parse template data
	lfsConfigDir := "/opt/lfs-user/config-dev"
	if k.appConfig.Production {
		lfsConfigDir = "/opt/lfs-user/config-prod"
	}
	obj, _, err := k.getRessourceFromFile("deployment-lfs.yaml",
		podCreateTemplateData{
			Username:           identifier,
			Db:                 "placeholder",
			LfsServiceEndpoint: k.appConfig.LfsServiceEndpoint,
			LfsConfigDir:       lfsConfigDir,
			Image:              k.appConfig.GetLfsImage(),
			ImageVersion:       k.appConfig.GetLfsImageVersion(),
			BaseName:           utils.GetEnvString("BASE_APP_NAME", "lfsx-web"),
			Namespace:          k.Namespace,
			IsPlaceholder:      true,
		},
	)
	if err != nil {
		return nil, err
	}

	// Convert to job and add job template label
	j := obj.(*batchv1.Job)
	j.Labels["placeholder"] = "true"

	// Create pod ressource
	job, err := k.Client.BatchV1().Jobs(k.Namespace).Create(context.TODO(), j, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create job: %s", err)
	}

	return job, err
}

// DeleteCompletedLfsPods removes all pods that are in a "completed" state
// and are having the
func (k *Kuber) DeleteCompletedLfsPods() error {

	// Filter condition for the container
	sel := metav1.LabelSelector{
		MatchLabels: map[string]string{
			"appGeneric": "lfs",
		},
	}

	// Execute the request
	pods, err := k.Client.CoreV1().Pods(k.Namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(&sel),
		FieldSelector: "status.phase=completed",
	})
	if err != nil {
		return err
	}

	// Loop through every pod and delete it
	for _, pod := range pods.Items {
		// Check if the pod was already scheduled to delete
		if pod.DeletionTimestamp != nil {
			logger.Debug("Trying to delete pod %q", pod.Name)
			if err := k.Client.CoreV1().Pods(k.Namespace).Delete(context.Background(), pod.Name, *metav1.NewDeleteOptions(5)); err != nil {
				logger.Debug("Failed to delete pod %q: %s", pod.Name, err)
			}
		}
	}

	return nil
}
