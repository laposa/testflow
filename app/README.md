# Test portal app

Using PHP Laravel + Sqlite.

Integrated with Github Actions and Auth.

Distributed as Github App - installed to GitHub organization with Test Portal instance hosted by Laposa. Similar setup to how <https://socket.dev/> works.

## Install

```bash
npm install
composer install
php artisan migrate # (or php artisan migrate:refresh)
```

## Create .env using 1Password cli

```bash
op inject -i .env.local.tpl -o .env
```

##  Run

```bash
npm run dev
php artisan serve
```

## Usage

- login via GitHub
- temp: link GitHub installation id in installations table:

```
sqlite3 database/database.sqlite 'insert into installations(id,installation_id,access_token,expires_at,repository_selection,created_at,updated_at) values (1,"51847197","ghs_SHEuPff5DjYqZiKxsjaqDt4mzMocr01blJXw","2024-06-17T16:51:08Z","selected", "2024-06-17 13:47:17","2024-06-17 15:51:08");' '.exit'
```
