package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	gap "github.com/muesli/go-app-paths"
)

func initTaskDir(path string) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return os.Mkdir(path, 0o770)
		}
		return err
	}
	return nil
}

func SetupPath() string {
	// get XDG paths
	scope := gap.NewScope(gap.User, "gauntlet")
	dirs, err := scope.DataDirs()
	if err != nil {
		log.Fatal(err)
	}
	// create the app base dir, if it doesn't exist
	var taskDir string
	if len(dirs) > 0 {
		taskDir = dirs[0]
	} else {
		taskDir, _ = os.UserHomeDir()
	}
	if err := initTaskDir(taskDir); err != nil {
		log.Fatal(err)
	}
	// fmt.Println(taskDir)
	return taskDir
}

func setupDB() *sql.DB {
	// Open a database connection
	path := SetupPath()
	db, err := sql.Open("sqlite3", filepath.Join(path, "tasks.db"))
	if err != nil {
		log.Fatal(err)
	}

	// Create the tasks table if it doesn't exist
	sqlStmt := `
	CREATE TABLE IF NOT EXISTS tasks (
		id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		description TEXT,
		score INTEGER
	);
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Fatalf("%q: %s\n", err, sqlStmt)
	}
	return db
}

func addTask(db *sql.DB, name, description string) {
	// Insert a new task with 0 score initially
	stmt, err := db.Prepare("INSERT INTO tasks(name, description, score) VALUES(?, ?, 0)")
	defer db.Close()
	if err != nil {
		log.Fatal(err)
	}
	_, err = stmt.Exec(name, description)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Task added!")
}

func calculateScore() int {
	// List of questions to ask
	questions := []string{
		"How urgent is this task? (1: Low, 2: Moderate, 3: High)",
		"Who will be affected by the completion of this task? (1: Small, 2: Team, 3: Organization)",
		"How long-lasting are the benefits of this task? (1: Low, 2: Moderate, 3: High)",
		"How much risk does this task mitigate? (1: Low, 2: Moderate, 3: High)",
		"How closely does this task align with key goals? (1: Low, 2: Moderate, 3: High)",
		"What opportunities are lost if this task is delayed? (1: Low, 2: Moderate, 3: High)",
		"How much effort vs reward for this task? (1: Low, 2: Moderate, 3: High)",
		"Does this task unblock other tasks? (1: Low, 2: Moderate, 3: High)",
	}

	totalScore := 0

	// Ask each question and gather responses
	for _, question := range questions {
		var response int
		fmt.Println(question)
		fmt.Scan(&response)
		totalScore += response
	}

	return totalScore
}

func assignScore(db *sql.DB) {
	// Select the latest task to assign a score
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	rows, err := tx.Query("SELECT id, name, description, score FROM tasks WHERE score = 0")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var id, score int
	var name, description string
	for rows.Next() {
		err := rows.Scan(&id, &name, &description, &score)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Scoring task: %s-%d\n", name, score)
		score := calculateScore()
		fmt.Println("score: ", score)

		// Update the task with the calculated score
		stmt, err := tx.Prepare("UPDATE tasks SET score = ? WHERE id = ?")
		if err != nil {
			log.Fatalf("Error preparing statement: %v", err)
		}
		defer stmt.Close()

		result, err := stmt.Exec(score, id)
		if err != nil {
			log.Fatalf("Error executing update: %v", err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			log.Fatalf("Error getting rows affected: %v", err)
		}
		fmt.Printf("Task scored! Rows affected: %d\n", rowsAffected)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		log.Fatal(err)
	}

}

func recommendTask(db *sql.DB) {
	var id int
	var name, description string
	err := db.QueryRow("SELECT id, name, description FROM tasks ORDER BY score DESC LIMIT 1").Scan(&id, &name, &description)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Next task to work on: %s - %s - %d\n", name, description, id)
}

func completeTask(db *sql.DB) {
	var id int
	fmt.Println("Enter the task ID to mark as done:")
	fmt.Scan(&id)

	stmt, err := db.Prepare("DELETE FROM tasks WHERE id = ?")
	if err != nil {
		log.Fatal(err)
	}
	_, err = stmt.Exec(id)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Task completed and removed!")
}

func main() {
	db := setupDB()
	defer db.Close()

	// Command-line arguments
	if len(os.Args) < 2 {
		fmt.Println("Usage: cli [add|score|recommend|done]")
		return
	}

	command := os.Args[1]

	switch command {
	case "add":
		if len(os.Args) < 4 {
			fmt.Println("Usage: cli add <task-name> <task-description>")
			return
		}
		name := os.Args[2]
		description := os.Args[3]
		addTask(db, name, description)

	case "score":
		assignScore(db)

	case "recommend":
		recommendTask(db)

	case "done":
		completeTask(db)

	default:
		fmt.Println("Unknown command. Use: add, score, recommend, or done")
	}
}

// go run main.go add "Write Go CLI" "Build a simple task manager in Go"
