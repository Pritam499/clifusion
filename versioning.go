// Command versioning and migration support
package cobra

import (
	"crypto/md5"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

type CommandSchema struct {
	Version    string
	CommandPath string
	Use        string
	Short      string
	Long       string
	Flags      map[string]string // flag name -> type:description
	Args       string
	SubCommands []string
	SchemaHash string
}

type MigrationFunc func(fromVersion, toVersion string, cmd *Command) error

type VersionManager struct {
	schemas    map[string]CommandSchema
	migrations map[string]MigrationFunc // "from-to" -> func
}

var GlobalVersionManager *VersionManager

func InitVersioning() {
	GlobalVersionManager = &VersionManager{
		schemas:    make(map[string]CommandSchema),
		migrations: make(map[string]MigrationFunc),
	}
}

func (vm *VersionManager) RegisterCommand(cmd *Command, version string) error {
	schema := vm.computeSchema(cmd, version)
	key := cmd.CommandPath() + ":" + version

	if existing, exists := vm.schemas[key]; exists {
		if existing.SchemaHash != schema.SchemaHash {
			return fmt.Errorf("schema mismatch for %s version %s", cmd.CommandPath(), version)
		}
	} else {
		vm.schemas[key] = schema
		// Store in DB if available
		if GlobalAnalyticsDB != nil {
			GlobalAnalyticsDB.StoreSchema(schema)
		}
	}

	return nil
}

func (vm *VersionManager) computeSchema(cmd *Command, version string) CommandSchema {
	schema := CommandSchema{
		Version:     version,
		CommandPath: cmd.CommandPath(),
		Use:         cmd.Use,
		Short:       cmd.Short,
		Long:        cmd.Long,
		Flags:       make(map[string]string),
		SubCommands: make([]string, 0),
	}

	// Collect flags
	cmd.Flags().VisitAll(func(f *flag.Flag) {
		desc := fmt.Sprintf("%s:%s", f.Value.Type(), f.Usage)
		schema.Flags[f.Name] = desc
	})

	// Collect subcommands
	for _, sub := range cmd.Commands() {
		schema.SubCommands = append(schema.SubCommands, sub.Name())
	}
	sort.Strings(schema.SubCommands)

	// Compute hash
	hashInput := fmt.Sprintf("%s|%s|%s|%s|%v|%v",
		schema.Use, schema.Short, schema.Long,
		schema.Args, schema.Flags, schema.SubCommands)
	schema.SchemaHash = fmt.Sprintf("%x", md5.Sum([]byte(hashInput)))

	return schema
}

func (vm *VersionManager) AddMigration(fromVersion, toVersion string, migration MigrationFunc) {
	key := fromVersion + "-" + toVersion
	vm.migrations[key] = migration
}

func (vm *VersionManager) MigrateCommand(cmd *Command, fromVersion, toVersion string) error {
	key := fromVersion + "-" + toVersion
	if migration, exists := vm.migrations[key]; exists {
		return migration(fromVersion, toVersion, cmd)
	}
	return fmt.Errorf("no migration path from %s to %s", fromVersion, toVersion)
}

func (vm *VersionManager) GetCompatibleVersions(cmdPath string) []string {
	versions := make([]string, 0)
	for key, schema := range vm.schemas {
		if strings.HasPrefix(key, cmdPath+":") {
			parts := strings.Split(key, ":")
			if len(parts) == 2 {
				versions = append(versions, parts[1])
			}
		}
	}
	return versions
}

func (vm *VersionManager) DetectVersionChange(cmd *Command, currentVersion string) (string, error) {
	key := cmd.CommandPath() + ":" + currentVersion
	if stored, exists := vm.schemas[key]; exists {
		computed := vm.computeSchema(cmd, currentVersion)
		if stored.SchemaHash != computed.SchemaHash {
			return "schema changed", nil
		}
	} else {
		return "new version", nil
	}
	return "", nil
}

// Database integration
func (a *AnalyticsDB) StoreSchema(schema CommandSchema) error {
	_, err := a.db.Exec(`
		INSERT OR REPLACE INTO command_schemas
		(command_path, version, use_desc, short_desc, long_desc, flags, args, subcommands, schema_hash)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, schema.CommandPath, schema.Version, schema.Use, schema.Short, schema.Long,
		fmt.Sprintf("%v", schema.Flags), schema.Args, strings.Join(schema.SubCommands, ","), schema.SchemaHash)
	return err
}

func (a *AnalyticsDB) GetStoredSchemas() ([]CommandSchema, error) {
	rows, err := a.db.Query(`SELECT command_path, version, use_desc, short_desc, long_desc, flags, args, subcommands, schema_hash FROM command_schemas`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schemas []CommandSchema
	for rows.Next() {
		var s CommandSchema
		var flagsStr, subsStr string
		err := rows.Scan(&s.CommandPath, &s.Version, &s.Use, &s.Short, &s.Long, &flagsStr, &s.Args, &subsStr, &s.SchemaHash)
		if err != nil {
			return nil, err
		}
		// Parse flags and subcommands if needed
		s.SubCommands = strings.Split(subsStr, ",")
		schemas = append(schemas, s)
	}
	return schemas, nil
}

func CreateVersioningCommand() *Command {
	cmd := &Command{
		Use:   "version",
		Short: "Manage command versions and migrations",
	}

	listCmd := &Command{
		Use:   "list <command>",
		Short: "List available versions for a command",
		Args:  ExactArgs(1),
		RunE: func(cmd *Command, args []string) error {
			if GlobalVersionManager == nil {
				return fmt.Errorf("versioning not initialized")
			}
			versions := GlobalVersionManager.GetCompatibleVersions(args[0])
			fmt.Printf("Versions for %s:\n", args[0])
			for _, v := range versions {
				fmt.Printf("  %s\n", v)
			}
			return nil
		},
	}

	migrateCmd := &Command{
		Use:   "migrate <command> <from> <to>",
		Short: "Migrate command from one version to another",
		Args:  ExactArgs(3),
		RunE: func(cmd *Command, args []string) error {
			if GlobalVersionManager == nil {
				return fmt.Errorf("versioning not initialized")
			}
			// Find the command
			root := cmd.Root()
			target, _, err := root.Find(strings.Fields(args[0]))
			if err != nil {
				return err
			}
			return GlobalVersionManager.MigrateCommand(target, args[1], args[2])
		},
	}

	checkCmd := &Command{
		Use:   "check",
		Short: "Check for schema changes",
		RunE: func(cmd *Command, args []string) error {
			if GlobalVersionManager == nil {
				return fmt.Errorf("versioning not initialized")
			}
			root := cmd.Root()
			return checkAllSchemas(root)
		},
	}

	cmd.AddCommand(listCmd, migrateCmd, checkCmd)
	return cmd
}

func checkAllSchemas(cmd *Command) error {
	if cmd.Version != "" {
		if change, err := GlobalVersionManager.DetectVersionChange(cmd, cmd.Version); err != nil {
			return err
		} else if change != "" {
			fmt.Printf("Command %s: %s\n", cmd.CommandPath(), change)
		}
	}
	for _, sub := range cmd.Commands() {
		checkAllSchemas(sub)
	}
	return nil
}

func init() {
	InitVersioning()
	// Create schema table
	if GlobalAnalyticsDB != nil {
		GlobalAnalyticsDB.db.Exec(`
			CREATE TABLE IF NOT EXISTS command_schemas (
				command_path TEXT,
				version TEXT,
				use_desc TEXT,
				short_desc TEXT,
				long_desc TEXT,
				flags TEXT,
				args TEXT,
				subcommands TEXT,
				schema_hash TEXT,
				PRIMARY KEY (command_path, version)
			)
		`)
	}
}