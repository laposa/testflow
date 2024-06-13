<html>
    <head>
        <title>{{ $title ?? 'Test Portal' }}</title>

        @vite(['resources/css/app.css', 'resources/js/app.js'])
    </head>
    <body>
        <x-header />
        <main>
            <div class="container">
                {{ $slot }}
            </div>
        </main>
        <x-footer />
    </body>
</html>
