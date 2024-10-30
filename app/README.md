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

## Generating Type hints

```bash
php artisan ide-helper:generate
php artisan ide-helper:models -N
```

## Linting & Formatting

Implemented using [Laravel Pint](https://laravel.com/docs/11.x/pint) and [Prettier](https://prettier.io/).

```bash
# lint only
./vendor/bin/pint  --test

# fix lint issues
./vendor/bin/pint

# format
npm run format
```

## Usage

- login via GitHub
