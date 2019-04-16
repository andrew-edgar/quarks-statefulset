package cmd

import (
	"fmt"
	"os"
	"regexp"
	"strconv"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"code.cloudfoundry.org/cf-operator/pkg/bosh/manifest"
)

// templateRenderCmd represents the template-render command
var templateRenderCmd = &cobra.Command{
	Use:   "template-render [flags]",
	Short: "Renders a bosh manifest",
	Long: `Renders a bosh manifest.

This will render a provided manifest instance-group
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		boshManifestPath := viper.GetString("bosh-manifest-path")
		jobsDir := viper.GetString("jobs-dir")
		instanceGroupName := viper.GetString("instance-group-name")

		specIndex := viper.GetInt("spec-index")
		if specIndex < 0 {
			// calculate index following the formula specified in
			// docs/rendering_templates.md
			azIndex := viper.GetInt("az-index")
			if azIndex < 0 {
				return fmt.Errorf("required parameter 'az-index' not set")
			}
			replicas := viper.GetInt("replicas")
			if replicas < 0 {
				return fmt.Errorf("required parameter 'replicas' not set")
			}
			podOrdinal := viper.GetInt("pod-ordinal")
			if podOrdinal < 0 {
				// Infer ordinal from hostname
				hostname, err := os.Hostname()
				r, _ := regexp.Compile(`-(\d+)(-|\.|\z)`)
				if err != nil {
					return errors.Wrap(err, "getting the hostname")
				}
				match := r.FindStringSubmatch(hostname)
				if len(match) < 2 {
					return fmt.Errorf("can not extract the pod ordinal from hostname '%s'", hostname)
				}
				podOrdinal, _ = strconv.Atoi(match[1])
			}

			specIndex = (azIndex-1)*replicas + podOrdinal
		}

		return manifest.RenderJobTemplates(boshManifestPath, jobsDir, "/var/vcap/jobs/", instanceGroupName, specIndex)
	},
}

func init() {
	rootCmd.AddCommand(templateRenderCmd)

	templateRenderCmd.Flags().StringP("bosh-manifest-path", "m", "", "path to the bosh manifest file")
	templateRenderCmd.Flags().StringP("jobs-dir", "j", "", "path to the jobs dir.")
	templateRenderCmd.Flags().StringP("instance-group-name", "g", "", "the instance-group name to render")
	templateRenderCmd.Flags().IntP("spec-index", "", -1, "index of the instance spec")
	templateRenderCmd.Flags().IntP("az-index", "", -1, "az index")
	templateRenderCmd.Flags().IntP("pod-ordinal", "", -1, "pod ordinal")
	templateRenderCmd.Flags().IntP("replicas", "", -1, "number of replicas")

	viper.BindPFlag("bosh-manifest-path", templateRenderCmd.Flags().Lookup("bosh-manifest-path"))
	viper.BindPFlag("jobs-dir", templateRenderCmd.Flags().Lookup("jobs-dir"))
	viper.BindPFlag("instance-group-name", templateRenderCmd.Flags().Lookup("instance-group-name"))
	viper.BindPFlag("az-index", templateRenderCmd.Flags().Lookup("az-index"))
	viper.BindPFlag("spec-index", templateRenderCmd.Flags().Lookup("spec-index"))
	viper.BindPFlag("pod-ordinal", templateRenderCmd.Flags().Lookup("pod-ordinal"))
	viper.BindPFlag("replicas", templateRenderCmd.Flags().Lookup("replicas"))

	argToEnv["bosh-manifest-path"] = "BOSH_MANIFEST_PATH"
	argToEnv["jobs-dir"] = "JOBS_DIR"
	argToEnv["instance-group-name"] = "INSTANCE_GROUP_NAME"
	argToEnv["docker-image-repository"] = "DOCKER_IMAGE_REPOSITORY"
	argToEnv["spec-index"] = "SPEC_INDEX"
	argToEnv["az-index"] = "AZ_INDEX"
	argToEnv["pod-ordinal"] = "POD_ORDINAL"
	argToEnv["replicas"] = "REPLICAS"

	for arg, env := range argToEnv {
		viper.BindEnv(arg, env)
	}
}
