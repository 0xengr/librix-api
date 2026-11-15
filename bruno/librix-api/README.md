# Librix API — Bruno collection

Open this folder as a collection in [Bruno](https://www.usebruno.com/):

1. Start the API: `make run` (it listens on `http://localhost:3000`).
2. In Bruno: **Open Collection** → select `bruno/librix-api`.
3. Pick the `local` environment (top-right selector).

Environment variables (edit in `environments/local.bru`):

| Variable  | Default                 | Meaning                          |
| --------- | ----------------------- | -------------------------------- |
| `baseUrl` | `http://localhost:3000` | API origin                       |
| `bookId`  | `1`                     | id of an existing book           |
| `userId`  | `1`                     | id of an existing user           |
| `days`    | `14`                    | loan length (`0` = server default) |

## Endpoints

| Method | Path                                | Body / Query | Success |
| ------ | ----------------------------------- | ------------ | ------- |
| GET    | `/`                                 | —            | 200     |
| GET    | `/api/v1/books`                     | —            | 200     |
| POST   | `/api/v1/books`                     | JSON         | 201     |
| GET    | `/api/v1/books/{id}`                | —            | 200     |
| PUT    | `/api/v1/books/{id}`                | JSON         | 200     |
| DELETE | `/api/v1/books/{id}`                | —            | 204     |
| POST   | `/api/v1/books/borrow`              | JSON         | 200     |
| POST   | `/api/v1/books/return`              | JSON         | 200     |
| GET    | `/api/v1/books/user/{id}/borrowed`  | —            | 200     |
| GET    | `/api/v1/books/overdue`             | —            | 200     |
| GET    | `/api/v1/users`                     | —            | 200     |
| POST   | `/api/v1/users`                     | JSON         | 201     |
| GET    | `/api/v1/users/{id}`                | —            | 200     |
| PUT    | `/api/v1/users/{id}`                | JSON         | 200     |
| DELETE | `/api/v1/users/{id}`                | —            | 204     |
| GET    | `/api/v1/users/{id}/history`        | —            | 200     |

## Suggested order

1. **Create User** → note the returned `id`, put it in `userId`.
2. **Create Book** → note the returned `id`, put it in `bookId`.
3. **Borrow Book** (marks the book unavailable).
4. **Get User Borrowed Books** / **Get Borrow History** / **Get Overdue Books**.
5. **Return Book** (marks the book available again).
6. **Get Borrow History** again to see `returned_at` filled in.
7. **Update / Delete** requests as needed.

## Notes

- Error responses always have the shape `{ "error": "message" }`.
- Book `description` and `year_of_publication` are nullable. On create/update,
  omitting them (`""` / `0`) stores `NULL`.
- `PUT /api/v1/books/{id}` requires both `title` and `author`.
- `PUT /api/v1/users/{id}` is a partial update: send only the fields to change.
- Borrowing requires an available book and an existing user, and the book is
  marked unavailable until it is returned.
- `days` must be `>= 0`; `0` means the default 14-day loan.
