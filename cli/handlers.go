package cli

import (
	"bufio"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"time"

	_ "embed"

	app "github.com/oriolf/simple-app"
	"github.com/oriolf/simple-app/db"
)

//go:embed Dockerfile
var dockerfile string

func DevServer(commandName string) func([]string) []string {
	return func([]string) []string {
		checkSum := ""
		for {
			newCheckSum, err := computeCodeChecksum()
			if err != nil {
				return []string{err.Error()}
			}

			if checkSum != newCheckSum {
				go func() {
					if err := restartDevServer(commandName); err != nil {
						fmt.Println("Could not restart dev server:", err)
					}
				}()
				checkSum = newCheckSum
			}

			time.Sleep(1 * time.Second)
		}
	}
}

func computeCodeChecksum() (string, error) {
	cmd := "ls *.go migrations/* static/* templates/* | xargs -I {} md5sum {}"
	output, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return "", fmt.Errorf("could not compute checksums: %w", err)
	}

	return string(output), nil
}

func restartDevServer(commandName string) error {
	fmt.Println("Restarting dev server...")

	binary := "simple-app-dev-server"
	if _, err := executa(fmt.Sprintf("killall %s || true", binary)); err != nil {
		return fmt.Errorf("could not kill previous process: %w", err)
	}

	if _, err := executa(fmt.Sprintf("go build -o %s", binary)); err != nil {
		return fmt.Errorf("could not build new binary: %w", err)
	}

	cmd := exec.Command("bash", "-c", fmt.Sprintf("./%s %s", binary, commandName))
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("could not get stderr: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("could not start new server: %w", err)
	}

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}
	}()

	cmd.Wait()

	return nil
}

func executa(cmd string) (string, error) {
	fmt.Printf("Executant «%s»\n", cmd)
	b, err := exec.Command("bash", "-c", cmd).Output()
	return string(b), err
}

func Deploy(serviceName string) func([]string) []string {
	f := func() error {
		repositoryUrl := os.Getenv("DOCKER_REPOSITORY_URL")
		serverAlias := os.Getenv("DEPLOYMENT_SERVER_ALIAS")
		deploymentsFolder := os.Getenv("DEPLOYMENTS_FOLDER")
		if repositoryUrl == "" || serverAlias == "" || deploymentsFolder == "" {
			return fmt.Errorf("Falten variables d'entorn: DOCKER_REPOSITORY_URL, DEPLOYMENT_SERVER_ALIAS o DEPLOYMENTS_FOLDER")
		}

		version, err := executa("git rev-parse --short HEAD")
		if err != nil {
			return fmt.Errorf("No s'ha pogut llegir la versió de git: %w", err)
		}

		if err := os.WriteFile("Dockerfile", []byte(dockerfile), 0666); err != nil {
			return fmt.Errorf("No s'ha pogut escriure el Dockerfile: %w", err)
		}

		version = strings.TrimSpace(version)
		imageName := fmt.Sprintf("%s/%s:%s", repositoryUrl, serviceName, version)
		if _, err := executa(fmt.Sprintf("docker build -t %s .", imageName)); err != nil {
			return fmt.Errorf("No s'ha pogut construir la imatge de docker: %w", err)
		}

		if err := os.Remove("Dockerfile"); err != nil {
			return fmt.Errorf("No s'ha esborrar el Dockerfile: %w", err)
		}

		if _, err := executa(fmt.Sprintf("docker push %s", imageName)); err != nil {
			return fmt.Errorf("No s'ha pogut enviar la imatge de docker: %w", err)
		}

		if _, err := executa(fmt.Sprintf("ssh %s 'docker pull %s'", serverAlias, imageName)); err != nil {
			return fmt.Errorf("No s'ha pogut rebre la imatge de docker: %w", err)
		}

		composeFile := "docker-compose.yml"
		command := fmt.Sprintf(
			"cd %s && python -m yq '.services.%s.image = \"%s\"' %s | sponge %s",
			deploymentsFolder, serviceName, imageName, composeFile, composeFile,
		)

		if _, err := executa(fmt.Sprintf(`ssh -t %s "%s"`, serverAlias, command)); err != nil {
			return fmt.Errorf("No s'ha pogut actualitzar la versió del docker compose: %w", err)
		}

		command = fmt.Sprintf("cd %s && docker compose up -d", deploymentsFolder)
		if _, err := executa(fmt.Sprintf(`ssh -t %s "%s"`, serverAlias, command)); err != nil {
			return fmt.Errorf("No s'ha pogut actualitzar el docker compose: %w", err)
		}

		return nil
	}

	return func(args []string) []string {
		if err := f(); err != nil {
			return []string{err.Error()}
		}
		return nil
	}
}

func Add[T app.Adder[T], PT app.Validator[T]](seeds ...func() *T) func([]string) []string {
	return func(args []string) []string {
		a := PT(new(T))
		if seeds != nil {
			a = seeds[0]()
		}

		params, err := parseCliParams(args)
		if err != nil {
			return []string{err.Error()}
		}

		if errors := a.Validate(params); errors.NotEmpty() {
			return append([]string{"There are errors in the parameters:"}, errors.FormatForCli()...)
		}

		var id uint
		f := func(tx *sql.Tx) (err error) {
			id, err = (*a).Add(tx)
			return err
		}
		if err := db.Transaction(f); err != nil {
			return []string{fmt.Sprintf("Could not add from cli: %s", app.TranslateError(err.Error()))}
		}

		return []string{fmt.Sprintf("Created entity with ID %d", id)}
	}
}

func GenerateTypescriptTypes(models ...any) func([]string) []string {
	return func([]string) []string {
		for _, model := range models {
			generateTypescriptTypes(model)
		}
		return nil
	}
}

func generateTypescriptTypes(model any) {
	t := reflect.TypeOf(model)
	s := "export type " + t.Name() + " = {\n"
	for _, field := range reflect.VisibleFields(t) {
		if tag := field.Tag.Get("json"); tag != "" && tag != "-" {
			s += "  " + tag + ": " + getTypescriptType(field.Type) + ";\n"
		}
	}
	s += "}\n"

	os.WriteFile(t.Name()+".ts", []byte(s), 0644)
}

func getTypescriptType(t reflect.Type) string {
	if app.InSlice(t.Name(), []string{"int", "uint", "float64"}) {
		return "number"
	}
	if t.Name() == "string" || t.Kind() == reflect.String {
		return "string"
	}
	if t.Kind() == reflect.Slice {
		return getTypescriptType(t.Elem()) + "[]"
	}
	if t.Kind() == reflect.Pointer {
		return getTypescriptType(t.Elem()) + "|null"
	}
	if t.Kind() == reflect.Struct {
		if app.InSlice(t.Name(), []string{"Date", "DateTime"}) {
			return "string"
		}
		return t.Name()
	}
	return "any"
}
