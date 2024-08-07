@php
    /** @var \Illuminate\Support\Collection|\App\Models\Session[] $sessions */
@endphp

<x-layout>
    <x-portal-section>
        <x-sessions.list :sessions="$sessions" />
    </x-portal-section>
</x-layout>
