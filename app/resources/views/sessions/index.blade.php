@php
    /** @var \Illuminate\Support\Collection|\App\Models\Session[] $sessions */
@endphp

<x-layout>
    <section>
        <h2>All Sessions</h2>
        <x-sessions.list :sessions="$sessions" />
    </section>
</x-layout>
