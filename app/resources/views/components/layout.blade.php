<!DOCTYPE html>
<html>

<head>
    <title>{{ config('app.name') ?? 'Test Portal' }}</title>

    @vite(['resources/css/app.css', 'resources/js/app.js'])
    @livewireStyles
</head>

<body>
    <x-header />
    <main>
        <div class="container">
            {{ $slot }}
        </div>
    </main>
    <x-footer />
    @livewireScripts
</body>

</html>
