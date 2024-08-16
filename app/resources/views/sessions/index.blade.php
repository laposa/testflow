@php
    /** @var \Illuminate\Support\Collection|\App\Models\Session[] $sessions */
@endphp

<x-layout>
    <section>
        <h2>All session</h2>
        <x-sessions.list :sessions="$sessions" />
    </section>
</x-layout>
