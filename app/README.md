# Test portal app

Using PHP Laravel + Sqlite.

Integrated with Github Actions and Auth.

Distributed as Github App - installed to GitHub organization with Test Portal instance hosted by Laposa. Similar setup to how https://socket.dev/ works.

## Install
```
npm install
composer install
php artisan migrate # (or php artisan migrate:refresh)
```

## Create .env using 1Password cli
```
op inject -i .env.local.tpl -o .env
```

## Run
```
npm run dev
php artisan serve
```
