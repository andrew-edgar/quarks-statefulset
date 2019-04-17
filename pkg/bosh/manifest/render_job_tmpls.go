package manifest

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	btg "github.com/viovanov/bosh-template-go"
	yaml "gopkg.in/yaml.v2"
)

// RenderJobTemplates will render templates for all jobs of the instance group
// https://bosh.io/docs/create-release/#job-specs
func RenderJobTemplates(boshManifestPath string, jobsDir string, jobsOutputDir string, instanceGroupName string, specIndex int) error {

	// Loading deployment manifest file
	resolvedYML, err := ioutil.ReadFile(boshManifestPath)
	if err != nil {
		return errors.Wrapf(err, "couldn't read manifest file %s", boshManifestPath)
	}
	boshManifest := Manifest{}
	err = yaml.Unmarshal(resolvedYML, &boshManifest)
	if err != nil {
		return errors.Wrapf(err, "failed to unmarshal deployment manifest %s", boshManifestPath)
	}

	// Loop over instancegroups
	for _, instanceGroup := range boshManifest.InstanceGroups {

		// Filter based on the instance group name
		if instanceGroup.Name != instanceGroupName {
			continue
		}

		// Render all files for all jobs included in this instance_group.
		for _, job := range instanceGroup.Jobs {
			jobInstanceLinks := []Link{}

			// Find job instance that's being rendered
			var currentJobInstance *JobInstance
			for _, instance := range job.Properties.BOSHContainerization.Instances {
				if instance.Index == specIndex {
					currentJobInstance = &instance
					break
				}
			}
			if currentJobInstance == nil {
				return fmt.Errorf("no instance found for spec index '%d'", specIndex)
			}

			// Loop over name and link
			for name, jobConsumersLink := range job.Properties.BOSHContainerization.Consumes {
				jobInstances := []JobInstance{}

				// Loop over instances of link
				for _, jobConsumerLinkInstance := range jobConsumersLink.Instances {
					jobInstances = append(jobInstances, JobInstance{
						Address: jobConsumerLinkInstance.Address,
						AZ:      jobConsumerLinkInstance.AZ,
						ID:      jobConsumerLinkInstance.ID,
						Index:   jobConsumerLinkInstance.Index,
						Name:    jobConsumerLinkInstance.Name,
					})
				}

				jobInstanceLinks = append(jobInstanceLinks, Link{
					Name:       name,
					Instances:  jobInstances,
					Properties: jobConsumersLink.Properties,
				})
			}

			jobSrcDir := filepath.Join(jobsDir, "jobs-src", job.Release, job.Name)
			jobMFFile := filepath.Join(jobSrcDir, "job.MF")
			jobMfBytes, err := ioutil.ReadFile(jobMFFile)
			if err != nil {
				return errors.Wrapf(err, "failed to read job spec file %s", jobMFFile)
			}

			jobSpec := JobSpec{}
			if err := yaml.Unmarshal([]byte(jobMfBytes), &jobSpec); err != nil {
				return errors.Wrapf(err, "failed to unmarshal job spec %s", jobMFFile)
			}

			// Loop over templates for rendering files
			for source, destination := range jobSpec.Templates {
				absDest := filepath.Join(jobsOutputDir, job.Name, destination)
				os.MkdirAll(filepath.Dir(absDest), 0755)

				properties := job.Properties.ToMap()

				renderPointer := btg.NewERBRenderer(
					&btg.EvaluationContext{
						Properties: properties,
					},

					&btg.InstanceInfo{
						Address: currentJobInstance.Address,
						AZ:      currentJobInstance.AZ,
						ID:      currentJobInstance.ID,
						Index:   string(currentJobInstance.Index),
						Name:    currentJobInstance.Name,
					},

					jobMFFile,
				)

				// Create the destination file
				absDestFile, err := os.Create(absDest)
				if err != nil {
					return err
				}
				defer absDestFile.Close()
				if err = renderPointer.Render(filepath.Join(jobSrcDir, "templates", source), absDestFile.Name()); err != nil {
					return err
				}
			}
		}
	}
	return nil
}