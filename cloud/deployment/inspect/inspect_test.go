package inspect

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/astronomer/astro-cli/astro-client"
	astro_mocks "github.com/astronomer/astro-cli/astro-client/mocks"
	testUtil "github.com/astronomer/astro-cli/pkg/testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	errGetDeployment = errors.New("test get deployment error")
	errMarshal       = errors.New("test error")
)

func errReturningYAMLMarshal(v interface{}) ([]byte, error) {
	return []byte{}, errMarshal
}

func errReturningJSONMarshal(v interface{}, prefix, indent string) ([]byte, error) {
	return []byte{}, errMarshal
}

func errorReturningDecode(input, output interface{}) error {
	return errMarshal
}

func restoreDecode(replace func(input, output interface{}) error) {
	decodeToStruct = replace
}

func restoreJSONMarshal(replace func(v interface{}, prefix, indent string) ([]byte, error)) {
	jsonMarshal = replace
}

func restoreYAMLMarshal(replace func(v interface{}) ([]byte, error)) {
	yamlMarshal = replace
}

func TestInspect(t *testing.T) {
	testUtil.InitTestConfig(testUtil.CloudPlatform)
	workspaceID := "test-ws-id"
	deploymentID := "test-deployment-id"
	deploymentName := "test-deployment-label"
	deploymentResponse := []astro.Deployment{
		{
			ID:          deploymentID,
			Label:       deploymentName,
			ReleaseName: "great-release-name",
			Workspace:   astro.Workspace{ID: workspaceID},
			Cluster: astro.Cluster{
				ID: "cluster-id",
				NodePools: []astro.NodePool{
					{
						ID:               "test-pool-id",
						IsDefault:        false,
						NodeInstanceType: "test-instance-type",
						CreatedAt:        time.Now(),
					},
					{
						ID:               "test-pool-id-1",
						IsDefault:        true,
						NodeInstanceType: "test-instance-type-1",
						CreatedAt:        time.Now(),
					},
				},
			},
			RuntimeRelease: astro.RuntimeRelease{Version: "6.0.0", AirflowVersion: "2.4.0"},
			DeploymentSpec: astro.DeploymentSpec{
				Executor: "CeleryExecutor",
				Scheduler: astro.Scheduler{
					AU:       5,
					Replicas: 3,
				},
				Webserver: astro.Webserver{URL: "some-url"},
			},
			WorkerQueues: []astro.WorkerQueue{
				{
					ID:                "test-wq-id",
					Name:              "default",
					IsDefault:         true,
					MaxWorkerCount:    130,
					MinWorkerCount:    12,
					WorkerConcurrency: 110,
					NodePoolID:        "test-pool-id",
				},
				{
					ID:                "test-wq-id-1",
					Name:              "test-queue-1",
					IsDefault:         false,
					MaxWorkerCount:    175,
					MinWorkerCount:    8,
					WorkerConcurrency: 150,
					NodePoolID:        "test-pool-id-1",
				},
			},
			UpdatedAt: time.Now(),
			Status:    "HEALTHY",
		},
		{
			ID:             "test-deployment-id-1",
			Label:          "test-deployment-label-1",
			RuntimeRelease: astro.RuntimeRelease{Version: "4.2.5"},
			DeploymentSpec: astro.DeploymentSpec{
				Scheduler: astro.Scheduler{
					AU:       5,
					Replicas: 3,
				},
			},
			WorkerQueues: []astro.WorkerQueue{
				{
					ID:                "test-wq-id-2",
					Name:              "test-queue-2",
					IsDefault:         false,
					MaxWorkerCount:    130,
					MinWorkerCount:    12,
					WorkerConcurrency: 110,
					NodePoolID:        "test-nodepool-id-2",
				},
				{
					ID:                "test-wq-id-3",
					Name:              "test-queue-3",
					IsDefault:         true,
					MaxWorkerCount:    175,
					MinWorkerCount:    8,
					WorkerConcurrency: 150,
					NodePoolID:        "test-nodepool-id-3",
				},
			},
		},
	}
	t.Run("prints a deployment in yaml format to stdout", func(t *testing.T) {
		out := new(bytes.Buffer)
		mockClient := new(astro_mocks.Client)
		mockClient.On("ListDeployments", mock.Anything, workspaceID).Return(deploymentResponse, nil).Once()
		err := Inspect(workspaceID, "", deploymentID, "yaml", mockClient, out, "")
		assert.NoError(t, err)
		assert.Contains(t, out.String(), deploymentResponse[0].ReleaseName)
		assert.Contains(t, out.String(), deploymentName)
		assert.Contains(t, out.String(), deploymentResponse[0].RuntimeRelease.Version)
		mockClient.AssertExpectations(t)
	})

	t.Run("prints a deployment in json format to stdout", func(t *testing.T) {
		out := new(bytes.Buffer)
		mockClient := new(astro_mocks.Client)
		mockClient.On("ListDeployments", mock.Anything, workspaceID).Return(deploymentResponse, nil).Once()
		err := Inspect(workspaceID, "", deploymentID, "json", mockClient, out, "")
		assert.NoError(t, err)
		assert.Contains(t, out.String(), deploymentResponse[0].ReleaseName)
		assert.Contains(t, out.String(), deploymentName)
		assert.Contains(t, out.String(), deploymentResponse[0].RuntimeRelease.Version)
		mockClient.AssertExpectations(t)
	})

	t.Run("prints a deployment's specific field to stdout", func(t *testing.T) {
		out := new(bytes.Buffer)
		mockClient := new(astro_mocks.Client)
		mockClient.On("ListDeployments", mock.Anything, workspaceID).Return(deploymentResponse, nil).Once()
		err := Inspect(workspaceID, "", deploymentID, "yaml", mockClient, out, "configuration.cluster_id")
		assert.NoError(t, err)
		assert.Contains(t, out.String(), deploymentResponse[0].Cluster.ID)
		mockClient.AssertExpectations(t)
	})

	t.Run("prompts for a deployment to inspect if no deployment name or id was provided", func(t *testing.T) {
		out := new(bytes.Buffer)
		mockClient := new(astro_mocks.Client)
		defer testUtil.MockUserInput(t, "1")() // selecting test-deployment-id
		mockClient.On("ListDeployments", mock.Anything, workspaceID).Return(deploymentResponse, nil).Once()
		err := Inspect(workspaceID, "", "", "yaml", mockClient, out, "")
		assert.NoError(t, err)
		assert.Contains(t, out.String(), deploymentName)
		mockClient.AssertExpectations(t)
	})

	t.Run("returns an error if listing deployment fails", func(t *testing.T) {
		out := new(bytes.Buffer)
		mockClient := new(astro_mocks.Client)
		mockClient.On("ListDeployments", mock.Anything, workspaceID).Return([]astro.Deployment{}, errGetDeployment).Once()
		err := Inspect(workspaceID, "", deploymentID, "yaml", mockClient, out, "")
		assert.ErrorIs(t, err, errGetDeployment)
		mockClient.AssertExpectations(t)
	})

	t.Run("returns an error if requested field is not found in deployment", func(t *testing.T) {
		out := new(bytes.Buffer)
		mockClient := new(astro_mocks.Client)
		mockClient.On("ListDeployments", mock.Anything, workspaceID).Return(deploymentResponse, nil).Once()
		err := Inspect(workspaceID, "", deploymentID, "yaml", mockClient, out, "no-exist-information")
		assert.ErrorIs(t, err, errKeyNotFound)
		assert.Equal(t, "", out.String())
		mockClient.AssertExpectations(t)
	})

	t.Run("returns an error if formatting deployment fails", func(t *testing.T) {
		out := new(bytes.Buffer)
		mockClient := new(astro_mocks.Client)
		originalMarshal := yamlMarshal
		yamlMarshal = errReturningYAMLMarshal
		defer restoreYAMLMarshal(originalMarshal)
		mockClient.On("ListDeployments", mock.Anything, workspaceID).Return(deploymentResponse, nil).Once()
		err := Inspect(workspaceID, "", deploymentID, "yaml", mockClient, out, "")
		assert.ErrorIs(t, err, errMarshal)
		mockClient.AssertExpectations(t)
	})

	t.Run("returns an error if getting context fails", func(t *testing.T) {
		testUtil.InitTestConfig(testUtil.ErrorReturningContext)
		out := new(bytes.Buffer)
		mockClient := new(astro_mocks.Client)
		err := Inspect(workspaceID, "", deploymentID, "yaml", mockClient, out, "")
		assert.ErrorContains(t, err, "no context set, have you authenticated to Astro or Astronomer Software? Run astro login and try again")
		mockClient.AssertExpectations(t)
	})
}

func TestGetDeploymentInspectInfo(t *testing.T) {
	sourceDeployment := astro.Deployment{
		ID:          "test-deployment-id",
		Label:       "test-deployment-label",
		Workspace:   astro.Workspace{ID: "test-ws-id"},
		ReleaseName: "great-release-name",
		Cluster: astro.Cluster{
			ID: "cluster-id",
			NodePools: []astro.NodePool{
				{
					ID:               "test-pool-id",
					IsDefault:        false,
					NodeInstanceType: "test-instance-type",
					CreatedAt:        time.Now(),
				},
				{
					ID:               "test-pool-id-1",
					IsDefault:        true,
					NodeInstanceType: "test-instance-type-1",
					CreatedAt:        time.Now(),
				},
			},
		},
		DagDeployEnabled: true,
		RuntimeRelease:   astro.RuntimeRelease{Version: "6.0.0", AirflowVersion: "2.4.0"},
		DeploymentSpec: astro.DeploymentSpec{
			Executor: "CeleryExecutor",
			Scheduler: astro.Scheduler{
				AU:       5,
				Replicas: 3,
			},
			Webserver: astro.Webserver{URL: "some-url"},
		},
		WorkerQueues: []astro.WorkerQueue{
			{
				ID:                "test-wq-id",
				Name:              "default",
				IsDefault:         true,
				MaxWorkerCount:    130,
				MinWorkerCount:    12,
				WorkerConcurrency: 110,
				NodePoolID:        "test-pool-id",
			},
			{
				ID:                "test-wq-id-1",
				Name:              "test-queue-1",
				IsDefault:         false,
				MaxWorkerCount:    175,
				MinWorkerCount:    8,
				WorkerConcurrency: 150,
				NodePoolID:        "test-pool-id-1",
			},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Status:    "HEALTHY",
	}

	t.Run("returns deployment metadata for the requested cloud deployment", func(t *testing.T) {
		var actualDeploymentMeta deploymentMetadata
		testUtil.InitTestConfig(testUtil.CloudPlatform)
		expectedCloudDomainURL := "cloud.astronomer.io/" + sourceDeployment.Workspace.ID +
			"/deployments/" + sourceDeployment.ID + "/analytics"
		expectedDeploymentMetadata := deploymentMetadata{
			DeploymentID:     sourceDeployment.ID,
			WorkspaceID:      sourceDeployment.Workspace.ID,
			ClusterID:        sourceDeployment.Cluster.ID,
			AirflowVersion:   sourceDeployment.RuntimeRelease.AirflowVersion,
			ReleaseName:      sourceDeployment.ReleaseName,
			DeploymentURL:    expectedCloudDomainURL,
			WebserverURL:     sourceDeployment.DeploymentSpec.Webserver.URL,
			CreatedAt:        sourceDeployment.CreatedAt,
			UpdatedAt:        sourceDeployment.UpdatedAt,
			Status:           sourceDeployment.Status,
			DagDeployEnabled: sourceDeployment.DagDeployEnabled,
		}
		rawDeploymentInfo, err := getDeploymentInfo(&sourceDeployment)
		assert.NoError(t, err)
		err = decodeToStruct(rawDeploymentInfo, &actualDeploymentMeta)
		assert.NoError(t, err)
		assert.Equal(t, expectedDeploymentMetadata, actualDeploymentMeta)
	})
	t.Run("returns deployment metadata for the requested local deployment", func(t *testing.T) {
		var actualDeploymentMeta deploymentMetadata
		testUtil.InitTestConfig(testUtil.LocalPlatform)
		expectedCloudDomainURL := "localhost:5000/" + sourceDeployment.Workspace.ID +
			"/deployments/" + sourceDeployment.ID + "/analytics"
		expectedDeploymentMetadata := deploymentMetadata{
			DeploymentID:     sourceDeployment.ID,
			WorkspaceID:      sourceDeployment.Workspace.ID,
			ClusterID:        sourceDeployment.Cluster.ID,
			ReleaseName:      sourceDeployment.ReleaseName,
			AirflowVersion:   sourceDeployment.RuntimeRelease.AirflowVersion,
			Status:           sourceDeployment.Status,
			CreatedAt:        sourceDeployment.CreatedAt,
			UpdatedAt:        sourceDeployment.UpdatedAt,
			DeploymentURL:    expectedCloudDomainURL,
			WebserverURL:     sourceDeployment.DeploymentSpec.Webserver.URL,
			DagDeployEnabled: sourceDeployment.DagDeployEnabled,
		}
		rawDeploymentInfo, err := getDeploymentInfo(&sourceDeployment)
		assert.NoError(t, err)
		err = decodeToStruct(rawDeploymentInfo, &actualDeploymentMeta)
		assert.NoError(t, err)
		assert.Equal(t, expectedDeploymentMetadata, actualDeploymentMeta)
	})
	t.Run("returns error if getting context fails", func(t *testing.T) {
		var actualDeploymentMeta deploymentMetadata
		// get an error from GetCurrentContext()
		testUtil.InitTestConfig(testUtil.ErrorReturningContext)
		expectedDeploymentMetadata := deploymentMetadata{}
		rawDeploymentInfo, err := getDeploymentInfo(&sourceDeployment)
		assert.ErrorContains(t, err, "no context set, have you authenticated to Astro or Astronomer Software? Run astro login and try again")
		err = decodeToStruct(rawDeploymentInfo, &actualDeploymentMeta)
		assert.NoError(t, err)
		assert.Equal(t, expectedDeploymentMetadata, actualDeploymentMeta)
	})
}

func TestGetDeploymentConfig(t *testing.T) {
	sourceDeployment := astro.Deployment{
		ID:          "test-deployment-id",
		Label:       "test-deployment-label",
		Description: "description",
		Workspace:   astro.Workspace{ID: "test-ws-id"},
		ReleaseName: "great-release-name",
		AlertEmails: []string{"email1", "email2"},
		Cluster: astro.Cluster{
			ID: "cluster-id",
			NodePools: []astro.NodePool{
				{
					ID:               "test-pool-id",
					IsDefault:        false,
					NodeInstanceType: "test-instance-type",
					CreatedAt:        time.Now(),
				},
				{
					ID:               "test-pool-id-1",
					IsDefault:        true,
					NodeInstanceType: "test-instance-type-1",
					CreatedAt:        time.Now(),
				},
			},
		},
		RuntimeRelease: astro.RuntimeRelease{Version: "6.0.0", AirflowVersion: "2.4.0"},
		DeploymentSpec: astro.DeploymentSpec{
			Executor: "CeleryExecutor",
			Scheduler: astro.Scheduler{
				AU:       5,
				Replicas: 3,
			},
			Webserver: astro.Webserver{URL: "some-url"},
			EnvironmentVariablesObjects: []astro.EnvironmentVariablesObject{
				{
					Key:       "foo",
					Value:     "bar",
					IsSecret:  false,
					UpdatedAt: "NOW",
				},
				{
					Key:       "bar",
					Value:     "baz",
					IsSecret:  true,
					UpdatedAt: "NOW+1",
				},
			},
		},
		WorkerQueues: []astro.WorkerQueue{
			{
				ID:                "test-wq-id",
				Name:              "default",
				IsDefault:         true,
				MaxWorkerCount:    130,
				MinWorkerCount:    12,
				WorkerConcurrency: 110,
				NodePoolID:        "test-pool-id",
			},
			{
				ID:                "test-wq-id-1",
				Name:              "test-queue-1",
				IsDefault:         false,
				MaxWorkerCount:    175,
				MinWorkerCount:    8,
				WorkerConcurrency: 150,
				NodePoolID:        "test-pool-id-1",
			},
		},
		UpdatedAt: time.Now(),
		Status:    "UNHEALTHY",
	}

	t.Run("returns deployment config for the requested cloud deployment", func(t *testing.T) {
		var actualDeploymentConfig deploymentConfig
		testUtil.InitTestConfig(testUtil.CloudPlatform)
		expectedDeploymentConfig := deploymentConfig{
			Name:           sourceDeployment.Label,
			Description:    sourceDeployment.Description,
			ClusterID:      sourceDeployment.Cluster.ID,
			RunTimeVersion: sourceDeployment.RuntimeRelease.Version,
			SchedulerAU:    sourceDeployment.DeploymentSpec.Scheduler.AU,
			SchedulerCount: sourceDeployment.DeploymentSpec.Scheduler.Replicas,
		}
		rawDeploymentConfig := getDeploymentConfig(&sourceDeployment)
		err := decodeToStruct(rawDeploymentConfig, &actualDeploymentConfig)
		assert.NoError(t, err)
		assert.Equal(t, expectedDeploymentConfig, actualDeploymentConfig)
	})
}

func TestGetPrintableDeployment(t *testing.T) {
	testUtil.InitTestConfig(testUtil.CloudPlatform)
	sourceDeployment := astro.Deployment{
		ID:          "test-deployment-id",
		Label:       "test-deployment-label",
		Description: "description",
		Workspace:   astro.Workspace{ID: "test-ws-id"},
		ReleaseName: "great-release-name",
		AlertEmails: []string{"email1", "email2"},
		Cluster: astro.Cluster{
			ID: "cluster-id",
			NodePools: []astro.NodePool{
				{
					ID:               "test-pool-id",
					IsDefault:        false,
					NodeInstanceType: "test-instance-type",
					CreatedAt:        time.Now(),
				},
				{
					ID:               "test-pool-id-1",
					IsDefault:        true,
					NodeInstanceType: "test-instance-type-1",
					CreatedAt:        time.Now(),
				},
			},
		},
		RuntimeRelease: astro.RuntimeRelease{Version: "6.0.0", AirflowVersion: "2.4.0"},
		DeploymentSpec: astro.DeploymentSpec{
			Executor: "CeleryExecutor",
			Scheduler: astro.Scheduler{
				AU:       5,
				Replicas: 3,
			},
			Webserver: astro.Webserver{URL: "some-url"},
			EnvironmentVariablesObjects: []astro.EnvironmentVariablesObject{
				{
					Key:       "foo",
					Value:     "bar",
					IsSecret:  false,
					UpdatedAt: "NOW",
				},
				{
					Key:       "bar",
					Value:     "baz",
					IsSecret:  true,
					UpdatedAt: "NOW+1",
				},
			},
		},
		WorkerQueues: []astro.WorkerQueue{
			{
				ID:                "test-wq-id",
				Name:              "default",
				IsDefault:         true,
				MaxWorkerCount:    130,
				MinWorkerCount:    12,
				WorkerConcurrency: 110,
				NodePoolID:        "test-pool-id",
			},
			{
				ID:                "test-wq-id-1",
				Name:              "test-queue-1",
				IsDefault:         false,
				MaxWorkerCount:    175,
				MinWorkerCount:    8,
				WorkerConcurrency: 150,
				NodePoolID:        "test-pool-id-1",
			},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Status:    "UNHEALTHY",
	}
	t.Run("returns a deployment map", func(t *testing.T) {
		info, _ := getDeploymentInfo(&sourceDeployment)
		config := getDeploymentConfig(&sourceDeployment)
		additional := getAdditional(&sourceDeployment)
		expectedDeployment := map[string]interface{}{
			"deployment": map[string]interface{}{
				"metadata":              info,
				"configuration":         config,
				"alert_emails":          additional["alert_emails"],
				"worker_queues":         additional["worker_queues"],
				"environment_variables": additional["environment_variables"],
			},
		}
		actualDeployment := getPrintableDeployment(info, config, additional)
		assert.Equal(t, expectedDeployment, actualDeployment)
	})
}

func TestGetAdditional(t *testing.T) {
	sourceDeployment := astro.Deployment{
		ID:          "test-deployment-id",
		Label:       "test-deployment-label",
		Description: "description",
		Workspace:   astro.Workspace{ID: "test-ws-id"},
		ReleaseName: "great-release-name",
		AlertEmails: []string{"email1", "email2"},
		Cluster: astro.Cluster{
			ID: "cluster-id",
			NodePools: []astro.NodePool{
				{
					ID:               "test-pool-id",
					IsDefault:        false,
					NodeInstanceType: "test-instance-type",
					CreatedAt:        time.Now(),
				},
				{
					ID:               "test-pool-id-1",
					IsDefault:        true,
					NodeInstanceType: "test-instance-type-1",
					CreatedAt:        time.Now(),
				},
			},
		},
		RuntimeRelease: astro.RuntimeRelease{Version: "6.0.0", AirflowVersion: "2.4.0"},
		DeploymentSpec: astro.DeploymentSpec{
			Executor: "CeleryExecutor",
			Scheduler: astro.Scheduler{
				AU:       5,
				Replicas: 3,
			},
			Webserver: astro.Webserver{URL: "some-url"},
			EnvironmentVariablesObjects: []astro.EnvironmentVariablesObject{
				{
					Key:       "foo",
					Value:     "bar",
					IsSecret:  false,
					UpdatedAt: time.Now().String(),
				},
				{
					Key:       "bar",
					Value:     "baz",
					IsSecret:  true,
					UpdatedAt: time.Now().String(),
				},
			},
		},
		WorkerQueues: []astro.WorkerQueue{
			{
				ID:                "test-wq-id",
				Name:              "default",
				IsDefault:         true,
				MaxWorkerCount:    130,
				MinWorkerCount:    12,
				WorkerConcurrency: 110,
				NodePoolID:        "test-pool-id",
			},
			{
				ID:                "test-wq-id-1",
				Name:              "test-queue-1",
				IsDefault:         false,
				MaxWorkerCount:    175,
				MinWorkerCount:    8,
				WorkerConcurrency: 150,
				NodePoolID:        "test-pool-id-1",
			},
		},
		UpdatedAt: time.Now(),
		Status:    "UNHEALTHY",
	}

	t.Run("returns alert emails, queues and variables for the requested deployment", func(t *testing.T) {
		var expectedAdditional, actualAdditional orderedPieces

		testUtil.InitTestConfig(testUtil.CloudPlatform)
		rawExpected := map[string]interface{}{
			"alert_emails":          sourceDeployment.AlertEmails,
			"worker_queues":         getQMap(sourceDeployment.WorkerQueues),
			"environment_variables": getVariablesMap(sourceDeployment.DeploymentSpec.EnvironmentVariablesObjects), // API only returns values when !EnvironmentVariablesObject.isSecret
		}
		rawAdditional := getAdditional(&sourceDeployment)
		err := decodeToStruct(rawAdditional, &actualAdditional)
		assert.NoError(t, err)
		err = decodeToStruct(rawExpected, &expectedAdditional)
		assert.NoError(t, err)
		assert.Equal(t, expectedAdditional, actualAdditional)
	})
}

func TestFormatPrintableDeployment(t *testing.T) {
	testUtil.InitTestConfig(testUtil.CloudPlatform)
	sourceDeployment := astro.Deployment{
		ID:          "test-deployment-id",
		Label:       "test-deployment-label",
		Description: "description",
		Workspace:   astro.Workspace{ID: "test-ws-id"},
		ReleaseName: "great-release-name",
		AlertEmails: []string{"email1", "email2"},
		Cluster: astro.Cluster{
			ID: "cluster-id",
			NodePools: []astro.NodePool{
				{
					ID:               "test-pool-id",
					IsDefault:        false,
					NodeInstanceType: "test-instance-type",
					CreatedAt:        time.Now(),
				},
				{
					ID:               "test-pool-id-1",
					IsDefault:        true,
					NodeInstanceType: "test-instance-type-1",
					CreatedAt:        time.Now(),
				},
			},
		},
		DagDeployEnabled: true,
		RuntimeRelease:   astro.RuntimeRelease{Version: "6.0.0", AirflowVersion: "2.4.0"},
		DeploymentSpec: astro.DeploymentSpec{
			Executor: "CeleryExecutor",
			Scheduler: astro.Scheduler{
				AU:       5,
				Replicas: 3,
			},
			Webserver: astro.Webserver{URL: "some-url"},
			EnvironmentVariablesObjects: []astro.EnvironmentVariablesObject{
				{
					Key:       "foo",
					Value:     "bar",
					IsSecret:  false,
					UpdatedAt: "NOW",
				},
				{
					Key:       "bar",
					Value:     "baz",
					IsSecret:  true,
					UpdatedAt: "NOW+1",
				},
			},
		},
		WorkerQueues: []astro.WorkerQueue{
			{
				ID:                "test-wq-id",
				Name:              "default",
				IsDefault:         true,
				MaxWorkerCount:    130,
				MinWorkerCount:    12,
				WorkerConcurrency: 110,
				NodePoolID:        "test-pool-id",
			},
			{
				ID:                "test-wq-id-1",
				Name:              "test-queue-1",
				IsDefault:         false,
				MaxWorkerCount:    175,
				MinWorkerCount:    8,
				WorkerConcurrency: 150,
				NodePoolID:        "test-pool-id-1",
			},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Status:    "UNHEALTHY",
	}
	var expectedPrintableDeployment []byte

	t.Run("returns a yaml formatted printable deployment", func(t *testing.T) {
		info, _ := getDeploymentInfo(&sourceDeployment)
		config := getDeploymentConfig(&sourceDeployment)
		additional := getAdditional(&sourceDeployment)

		printableDeployment := map[string]interface{}{
			"deployment": map[string]interface{}{
				"metadata":              info,
				"configuration":         config,
				"alert_emails":          additional["alert_emails"],
				"worker_queues":         additional["worker_queues"],
				"environment_variables": additional["environment_variables"],
			},
		}
		expectedDeployment := `deployment:
    environment_variables:
        - is_secret: false
          key: foo
          updated_at: NOW
          value: bar
        - is_secret: true
          key: bar
          updated_at: NOW+1
          value: baz
    configuration:
        name: test-deployment-label
        description: description
        runtime_version: 6.0.0
        scheduler_au: 5
        scheduler_count: 3
        cluster_id: cluster-id
    worker_queues:
        - name: default
          id: test-wq-id
          is_default: true
          max_worker_count: 130
          min_worker_count: 12
          worker_concurrency: 110
          node_pool_id: test-pool-id
        - name: test-queue-1
          id: test-wq-id-1
          is_default: false
          max_worker_count: 175
          min_worker_count: 8
          worker_concurrency: 150
          node_pool_id: test-pool-id-1
    metadata:
        deployment_id: test-deployment-id
        workspace_id: test-ws-id
        cluster_id: cluster-id
        release_name: great-release-name
        airflow_version: 2.4.0
        dag_deploy_enabled: true
        status: UNHEALTHY
        created_at: 2022-11-17T13:25:55.275697-08:00
        updated_at: 2022-11-17T13:25:55.275697-08:00
        deployment_url: cloud.astronomer.io/test-ws-id/deployments/test-deployment-id/analytics
        webserver_url: some-url
    alert_emails:
        - email1
        - email2
`
		var orderedAndTaggedDeployment, unorderedDeployment formattedDeployment
		actualPrintableDeployment, err := formatPrintableDeployment("", printableDeployment)
		assert.NoError(t, err)
		// testing we get valid yaml
		err = yaml.Unmarshal(actualPrintableDeployment, &orderedAndTaggedDeployment)
		assert.NoError(t, err)
		// update time and create time are not equal here so can not do equality check
		assert.NotEqual(t, expectedDeployment, string(actualPrintableDeployment), "tag and order should match")

		unordered, err := yaml.Marshal(printableDeployment)
		assert.NoError(t, err)
		err = yaml.Unmarshal(unordered, &unorderedDeployment)
		assert.NoError(t, err)
		// testing the structs are equal regardless of order
		assert.Equal(t, orderedAndTaggedDeployment, unorderedDeployment, "structs should match")
		// testing the order is not equal
		assert.NotEqual(t, string(unordered), string(actualPrintableDeployment), "order should not match")
	})
	t.Run("returns a json formatted printable deployment", func(t *testing.T) {
		info, _ := getDeploymentInfo(&sourceDeployment)
		config := getDeploymentConfig(&sourceDeployment)
		additional := getAdditional(&sourceDeployment)
		printableDeployment := map[string]interface{}{
			"deployment": map[string]interface{}{
				"metadata":              info,
				"configuration":         config,
				"alert_emails":          additional["alert_emails"],
				"worker_queues":         additional["worker_queues"],
				"environment_variables": additional["environment_variables"],
			},
		}
		expectedDeployment := `{
    "deployment": {
        "environment_variables": [
            {
                "is_secret": false,
                "key": "foo",
                "updated_at": "NOW",
                "value": "bar"
            },
            {
                "is_secret": true,
                "key": "bar",
                "updated_at": "NOW+1",
                "value": "baz"
            }
        ],
        "configuration": {
            "name": "test-deployment-label",
            "description": "description",
            "runtime_version": "6.0.0",
            "scheduler_au": 5,
            "scheduler_count": 3,
            "cluster_id": "cluster-id"
        },
        "worker_queues": [
            {
                "name": "default",
                "id": "test-wq-id",
                "is_default": true,
                "max_worker_count": 130,
                "min_worker_count": 12,
                "worker_concurrency": 110,
                "node_pool_id": "test-pool-id"
            },
            {
                "name": "test-queue-1",
                "id": "test-wq-id-1",
                "is_default": false,
                "max_worker_count": 175,
                "min_worker_count": 8,
                "worker_concurrency": 150,
                "node_pool_id": "test-pool-id-1"
            }
        ],
        "metadata": {
            "deployment_id": "test-deployment-id",
            "workspace_id": "test-ws-id",
            "cluster_id": "cluster-id",
            "release_name": "great-release-name",
            "airflow_version": "2.4.0",
            "dag_deploy_enabled": true,
            "status": "UNHEALTHY",
            "created_at": "2022-11-17T12:26:45.362983-08:00",
            "updated_at": "2022-11-17T12:26:45.362983-08:00",
            "deployment_url": "cloud.astronomer.io/test-ws-id/deployments/test-deployment-id/analytics",
            "webserver_url": "some-url"
        },
        "alert_emails": [
            "email1",
            "email2"
        ]
    }
}`
		var orderedAndTaggedDeployment, unorderedDeployment formattedDeployment
		actualPrintableDeployment, err := formatPrintableDeployment("json", printableDeployment)
		assert.NoError(t, err)
		// testing we get valid json
		err = json.Unmarshal(actualPrintableDeployment, &orderedAndTaggedDeployment)
		assert.NoError(t, err)
		// update time and create time are not equal here so can not do equality check
		assert.NotEqual(t, expectedDeployment, string(actualPrintableDeployment), "tag and order should match")

		unordered, err := json.MarshalIndent(printableDeployment, "", "    ")
		assert.NoError(t, err)
		err = json.Unmarshal(unordered, &unorderedDeployment)
		assert.NoError(t, err)
		// testing the structs are equal regardless of order
		assert.Equal(t, orderedAndTaggedDeployment, unorderedDeployment, "structs should match")
		// testing the order is not equal
		assert.NotEqual(t, string(unordered), string(actualPrintableDeployment), "order should not match")
	})
	t.Run("returns an error if decoding to struct fails", func(t *testing.T) {
		originalDecode := decodeToStruct
		decodeToStruct = errorReturningDecode
		defer restoreDecode(originalDecode)
		info, _ := getDeploymentInfo(&sourceDeployment)
		config := getDeploymentConfig(&sourceDeployment)
		additional := getAdditional(&sourceDeployment)
		expectedPrintableDeployment = []byte{}
		actualPrintableDeployment, err := formatPrintableDeployment("", getPrintableDeployment(info, config, additional))
		assert.ErrorIs(t, err, errMarshal)
		assert.Contains(t, string(actualPrintableDeployment), string(expectedPrintableDeployment))
	})
	t.Run("returns an error if marshaling yaml fails", func(t *testing.T) {
		originalMarshal := yamlMarshal
		yamlMarshal = errReturningYAMLMarshal
		defer restoreYAMLMarshal(originalMarshal)
		info, _ := getDeploymentInfo(&sourceDeployment)
		config := getDeploymentConfig(&sourceDeployment)
		additional := getAdditional(&sourceDeployment)
		expectedPrintableDeployment = []byte{}
		actualPrintableDeployment, err := formatPrintableDeployment("", getPrintableDeployment(info, config, additional))
		assert.ErrorIs(t, err, errMarshal)
		assert.Contains(t, string(actualPrintableDeployment), string(expectedPrintableDeployment))
	})
	t.Run("returns an error if marshaling json fails", func(t *testing.T) {
		originalMarshal := jsonMarshal
		jsonMarshal = errReturningJSONMarshal
		defer restoreJSONMarshal(originalMarshal)
		info, _ := getDeploymentInfo(&sourceDeployment)
		config := getDeploymentConfig(&sourceDeployment)
		additional := getAdditional(&sourceDeployment)
		expectedPrintableDeployment = []byte{}
		actualPrintableDeployment, err := formatPrintableDeployment("json", getPrintableDeployment(info, config, additional))
		assert.ErrorIs(t, err, errMarshal)
		assert.Contains(t, string(actualPrintableDeployment), string(expectedPrintableDeployment))
	})
}

func TestGetSpecificField(t *testing.T) {
	sourceDeployment := astro.Deployment{
		ID:          "test-deployment-id",
		Label:       "test-deployment-label",
		Description: "description",
		Workspace:   astro.Workspace{ID: "test-ws-id"},
		ReleaseName: "great-release-name",
		AlertEmails: []string{"email1", "email2"},
		Cluster: astro.Cluster{
			ID: "cluster-id",
			NodePools: []astro.NodePool{
				{
					ID:               "test-pool-id",
					IsDefault:        false,
					NodeInstanceType: "test-instance-type",
					CreatedAt:        time.Now(),
				},
				{
					ID:               "test-pool-id-1",
					IsDefault:        true,
					NodeInstanceType: "test-instance-type-1",
					CreatedAt:        time.Now(),
				},
			},
		},
		RuntimeRelease: astro.RuntimeRelease{Version: "6.0.0", AirflowVersion: "2.4.0"},
		DeploymentSpec: astro.DeploymentSpec{
			Executor: "CeleryExecutor",
			Scheduler: astro.Scheduler{
				AU:       5,
				Replicas: 3,
			},
			Webserver: astro.Webserver{URL: "some-url"},
			EnvironmentVariablesObjects: []astro.EnvironmentVariablesObject{
				{
					Key:       "foo",
					Value:     "bar",
					IsSecret:  false,
					UpdatedAt: "NOW",
				},
				{
					Key:       "bar",
					Value:     "baz",
					IsSecret:  true,
					UpdatedAt: "NOW+1",
				},
			},
		},
		WorkerQueues: []astro.WorkerQueue{
			{
				ID:                "test-wq-id",
				Name:              "default",
				IsDefault:         true,
				MaxWorkerCount:    130,
				MinWorkerCount:    12,
				WorkerConcurrency: 110,
				NodePoolID:        "test-pool-id",
			},
			{
				ID:                "test-wq-id-1",
				Name:              "test-queue-1",
				IsDefault:         false,
				MaxWorkerCount:    175,
				MinWorkerCount:    8,
				WorkerConcurrency: 150,
				NodePoolID:        "test-pool-id-1",
			},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Status:    "UNHEALTHY",
	}
	testUtil.InitTestConfig(testUtil.CloudPlatform)
	info, _ := getDeploymentInfo(&sourceDeployment)
	config := getDeploymentConfig(&sourceDeployment)
	additional := getAdditional(&sourceDeployment)
	t.Run("returns a value if key is found in deployment.metadata", func(t *testing.T) {
		requestedField := "metadata.workspace_id"
		printableDeployment := map[string]interface{}{
			"deployment": map[string]interface{}{
				"metadata":              info,
				"configuration":         config,
				"alert_emails":          additional["alert_emails"],
				"worker_queues":         additional["worker_queues"],
				"environment_variables": additional["environment_variables"],
			},
		}
		actual, err := getSpecificField(printableDeployment, requestedField)
		assert.NoError(t, err)
		assert.Equal(t, sourceDeployment.Workspace.ID, actual)
	})
	t.Run("returns a value if key is found in deployment.configuration", func(t *testing.T) {
		requestedField := "configuration.scheduler_count"
		printableDeployment := map[string]interface{}{
			"deployment": map[string]interface{}{
				"metadata":              info,
				"configuration":         config,
				"alert_emails":          additional["alert_emails"],
				"worker_queues":         additional["worker_queues"],
				"environment_variables": additional["environment_variables"],
			},
		}
		actual, err := getSpecificField(printableDeployment, requestedField)
		assert.NoError(t, err)
		assert.Equal(t, sourceDeployment.DeploymentSpec.Scheduler.Replicas, actual)
	})
	t.Run("returns a value if key is alert_emails", func(t *testing.T) {
		requestedField := "alert_emails"
		printableDeployment := map[string]interface{}{
			"deployment": map[string]interface{}{
				"metadata":              info,
				"configuration":         config,
				"alert_emails":          additional["alert_emails"],
				"worker_queues":         additional["worker_queues"],
				"environment_variables": additional["environment_variables"],
			},
		}
		actual, err := getSpecificField(printableDeployment, requestedField)
		assert.NoError(t, err)
		assert.Equal(t, sourceDeployment.AlertEmails, actual)
	})
	t.Run("returns a value if key is environment_variables", func(t *testing.T) {
		requestedField := "environment_variables"
		printableDeployment := map[string]interface{}{
			"deployment": map[string]interface{}{
				"metadata":              info,
				"configuration":         config,
				"alert_emails":          additional["alert_emails"],
				"worker_queues":         additional["worker_queues"],
				"environment_variables": additional["environment_variables"],
			},
		}
		actual, err := getSpecificField(printableDeployment, requestedField)
		assert.NoError(t, err)
		assert.Equal(t, getVariablesMap(sourceDeployment.DeploymentSpec.EnvironmentVariablesObjects), actual)
	})
	t.Run("returns a value if key is worker_queues", func(t *testing.T) {
		requestedField := "worker_queues"
		printableDeployment := map[string]interface{}{
			"deployment": map[string]interface{}{
				"metadata":              info,
				"configuration":         config,
				"alert_emails":          additional["alert_emails"],
				"worker_queues":         additional["worker_queues"],
				"environment_variables": additional["environment_variables"],
			},
		}
		actual, err := getSpecificField(printableDeployment, requestedField)
		assert.NoError(t, err)
		assert.Equal(t, getQMap(sourceDeployment.WorkerQueues), actual)
	})
	t.Run("returns a value if key is metadata", func(t *testing.T) {
		requestedField := "metadata"
		printableDeployment := map[string]interface{}{
			"deployment": map[string]interface{}{
				"metadata":              info,
				"configuration":         config,
				"alert_emails":          additional["alert_emails"],
				"worker_queues":         additional["worker_queues"],
				"environment_variables": additional["environment_variables"],
			},
		}
		actual, err := getSpecificField(printableDeployment, requestedField)
		assert.NoError(t, err)
		assert.Equal(t, info, actual)
	})
	t.Run("returns value regardless of upper or lower case key", func(t *testing.T) {
		requestedField := "Configuration.Cluster_ID"
		printableDeployment := map[string]interface{}{
			"deployment": map[string]interface{}{
				"metadata":              info,
				"configuration":         config,
				"alert_emails":          additional["alert_emails"],
				"worker_queues":         additional["worker_queues"],
				"environment_variables": additional["environment_variables"],
			},
		}
		actual, err := getSpecificField(printableDeployment, requestedField)
		assert.NoError(t, err)
		assert.Equal(t, sourceDeployment.Cluster.ID, actual)
	})
	t.Run("returns error if no value is found", func(t *testing.T) {
		printableDeployment := map[string]interface{}{
			"deployment": map[string]interface{}{
				"metadata":              info,
				"configuration":         config,
				"alert_emails":          additional["alert_emails"],
				"worker_queues":         additional["worker_queues"],
				"environment_variables": additional["astronomer_variables"],
			},
		}
		requestedField := "does-not-exist"
		actual, err := getSpecificField(printableDeployment, requestedField)
		assert.ErrorContains(t, err, "requested key "+requestedField+" not found in deployment")
		assert.Equal(t, nil, actual)
	})
	t.Run("returns error if incorrect field is requested", func(t *testing.T) {
		printableDeployment := map[string]interface{}{
			"deployment": map[string]interface{}{
				"metadata":              info,
				"configuration":         config,
				"alert_emails":          additional["alert_emails"],
				"worker_queues":         additional["worker_queues"],
				"environment_variables": additional["astronomer_variables"],
			},
		}
		requestedField := "configuration.does-not-exist"
		actual, err := getSpecificField(printableDeployment, requestedField)
		assert.ErrorIs(t, err, errKeyNotFound)
		assert.Equal(t, nil, actual)
	})
}