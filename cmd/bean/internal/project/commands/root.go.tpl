{{ .Copyright }}
package commands

import (
	"log"
	"os"

	"{{ .PkgName }}/commands/gopher"

	"github.com/retail-ai-inc/bean/v2"
	"github.com/retail-ai-inc/bean/v2/helpers"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "{{ .PkgName }} command [args...]",
	Short: "",
	Long:  "",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	
	// Clean up bean resources before exiting.
	defer bean.Cleanup()
	
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.Version = helpers.CurrVersion()

	viper.AddConfigPath(".")
	viper.SetConfigType("json")
	viper.SetConfigName("env")

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalln(err)
	}

	// IMPORTANT: Unmarshal the env.json into global BeanConfig object.
	if err := viper.Unmarshal(&bean.BeanConfig); err != nil {
		log.Fatalln(err)
	}

	rootCmd.AddCommand(gopher.GopherCmd)
}
