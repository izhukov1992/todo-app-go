package repository

import (
	"errors"
	"fmt"
	"strings"
	"todo-app"

	"github.com/jmoiron/sqlx"
)

type TodoItemPostgres struct {
	db *sqlx.DB
}

func NewTodoItemPostgres(db *sqlx.DB) *TodoItemPostgres {
	return &TodoItemPostgres{db: db}
}

func (r *TodoItemPostgres) Create(listId int, item todo.TodoItem) (int, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return 0, err
	}

	var itemId int
	createItemQuery := fmt.Sprintf("INSERT INTO %s (title, description) VALUES ($1, $2) RETURNING id", todoItemsTable)

	row := tx.QueryRow(createItemQuery, item.Title, item.Description)
	err = row.Scan(&itemId)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	createListItemsQuery := fmt.Sprintf("INSERT INTO %s (item_id, list_id) VALUES ($1, $2)", listsItemsTable)
	_, err = tx.Exec(createListItemsQuery, itemId, listId)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	return itemId, tx.Commit()
}

func (r *TodoItemPostgres) GetAll(userId, listId int) ([]todo.TodoItem, error) {
	var items []todo.TodoItem
	query := fmt.Sprintf("SELECT ti.id, ti.title, ti.description, ti.done FROM %s ti INNER JOIN %s li ON ti.id=li.item_id INNER JOIN %s ul ON li.list_id=ul.list_id WHERE li.list_id=$1 AND ul.user_id=$2", todoItemsTable, listsItemsTable, usersListsTable)
	err := r.db.Select(&items, query, listId, userId)
	return items, err
}

func (r *TodoItemPostgres) GetById(userId, itemId int) (todo.TodoItem, error) {
	var item todo.TodoItem
	query := fmt.Sprintf("SELECT ti.id, ti.title, ti.description, ti.done FROM %s ti INNER JOIN %s li ON ti.id=li.item_id INNER JOIN %s ul ON li.list_id=ul.list_id WHERE ti.id=$1 AND ul.user_id=$2", todoItemsTable, listsItemsTable, usersListsTable)
	err := r.db.Get(&item, query, itemId, userId)
	return item, err
}

func (r *TodoItemPostgres) Delete(userId, itemId int) error {
	query := fmt.Sprintf("DELETE FROM %s ti USING %s li, %s ul WHERE ti.id=li.item_id AND li.list_id=ul.list_id AND ti.id=$1 AND ul.user_id=$2", todoItemsTable, listsItemsTable, usersListsTable)
	res, err := r.db.Exec(query, itemId, userId)
	if err != nil {
		return err
	}

	num, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if num == 0 {
		return errors.New("item not found")
	}

	return err
}

func (r *TodoItemPostgres) Update(userId, itemId int, item todo.UpdateItemInput) error {
	setValues := make([]string, 0)
	args := make([]interface{}, 0)
	argId := 1

	if item.Title != nil {
		setValues = append(setValues, fmt.Sprintf("title=$%d", argId))
		args = append(args, *item.Title)
		argId++
	}
	if item.Description != nil {
		setValues = append(setValues, fmt.Sprintf("description=$%d", argId))
		args = append(args, *item.Description)
		argId++
	}
	if item.Done != nil {
		setValues = append(setValues, fmt.Sprintf("done=$%d", argId))
		args = append(args, *item.Done)
		argId++
	}
	args = append(args, userId, itemId)

	querySet := strings.Join(setValues, ", ")
	query := fmt.Sprintf("UPDATE %s ti SET %s FROM %s li, %s ul WHERE ti.id=li.item_id AND li.list_id=ul.list_id AND ul.user_id=$%d AND ti.id=$%d", todoItemsTable, querySet, listsItemsTable, usersListsTable, argId, argId+1)
	res, err := r.db.Exec(query, args...)
	if err != nil {
		return err
	}

	num, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if num == 0 {
		return errors.New("item not found")
	}

	return err
}
