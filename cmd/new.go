package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/AlecAivazis/survey/v2"

	"github.com/spf13/cobra"
)

const (
	repoURL = "https://github.com/Q1mi/gin-layout-base.git"
)

type Project struct {
	ProjectName string `survey:"name"`
	FolderName  string
}

var NewCmd = &cobra.Command{
	Use:     "new",
	Example: "xxx new project-name",
	Short:   "create a new project.",
	Long:    `create a new project with gin-layout-base.`,
	Run:     run,
}

func NewProject(projectName string) *Project {
	return &Project{
		ProjectName: projectName,
		FolderName:  filepath.Base(projectName), //  eq: github.com/xx/xxx -> xxx
	}
}

func run(cmd *cobra.Command, args []string) {
	p := NewProject(args[0])
	if len(args) == 0 {
		fmt.Println("need project name")
		return
	}

	// clone repo
	yes, err := p.cloneRepo()
	if err != nil || !yes {
		return
	}

	// replace package name
	err = p.replacePackageName()
	if err != nil || !yes {
		return
	}

	// go mod tidy
	err = p.modTidy()
	if err != nil || !yes {
		return
	}
	p.rmGit()
	fmt.Printf("🎉 🎉 🎉 Project \u001B[36m%s\u001B[0m created successfully!\n\n", p.ProjectName)
	fmt.Printf("Now run:\n\n")
	fmt.Printf("› \033[36mcd %s \033[0m\n", p.FolderName)
	fmt.Printf("› \033[36mgo run cmd/server/main.go\033[0m\n\n")
}

func (p *Project) cloneRepo() (bool, error) {
	// 1.检查目录是否已存在
	stat, _ := os.Stat(p.ProjectName)
	if stat != nil {
		var overwrite = false

		prompt := &survey.Confirm{
			Message: fmt.Sprintf("Folder %s already exists, do you want to overwrite it?", p.ProjectName),
			Help:    "Remove old project and create new project.",
		}
		err := survey.AskOne(prompt, &overwrite)
		if err != nil {
			return false, err
		}
		if !overwrite {
			return false, nil
		}
		err = os.RemoveAll(p.ProjectName)
		if err != nil {
			fmt.Println("remove old project error: ", err)
			return false, err
		}
	}

	fmt.Printf("git clone %s\n", repoURL)
	cmd := exec.Command("git", "clone", repoURL, p.ProjectName)
	_, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("git clone %s error: %s\n", repoURL, err)
		return false, err
	}
	return true, nil
}

func (p *Project) replacePackageName() error {
	moduleName := p.getModuleName()
	if len(moduleName) == 0 {
		return fmt.Errorf("get module name error")
	}
	err := p.replaceFiles(moduleName)
	if err != nil {
		return err
	}

	cmd := exec.Command("go", "mod", "edit", "-module", p.ProjectName)
	cmd.Dir = p.FolderName
	_, err = cmd.CombinedOutput()
	if err != nil {
		fmt.Println("go mod edit error: ", err)
		return err
	}
	return nil
}
func (p *Project) modTidy() error {
	fmt.Println("go mod tidy")
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = p.FolderName
	if err := cmd.Run(); err != nil {
		fmt.Println("go mod tidy error: ", err)
		return err
	}
	return nil
}
func (p *Project) rmGit() {
	os.RemoveAll(filepath.Join(p.FolderName, ".git"))
}

func (p *Project) replaceFiles(old string) error {
	err := filepath.Walk(p.FolderName, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		newData := bytes.ReplaceAll(data, []byte(old), []byte(p.ProjectName))
		if err := os.WriteFile(path, newData, 0644); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		fmt.Println("walk file do replace error: ", err)
		return err
	}
	return nil
}

// getModuleName 从 go.mod 中获取 module name
func (p *Project) getModuleName() string {
	modFile, err := os.Open(filepath.Join(p.FolderName, "go.mod"))
	if err != nil {
		fmt.Println("go.mod does not exist", err)
		return ""
	}
	defer modFile.Close()

	var moduleName string
	_, err = fmt.Fscanf(modFile, "module %s", &moduleName)
	if err != nil {
		fmt.Println("read go mod error: ", err)
		return ""
	}
	return moduleName
}
