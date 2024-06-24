<x-layout>
    <h1>{{ $session->name }}</h1>
    <livewire:runs.list-runs :session="$session" />

    <div></div>
    <form action="{{ route('session.runs.store', ['session' => $session]) }}" method="POST">
        @csrf
        <button type="submit" class="button align-right">Run all tests</button>
    </form>

    <x-portal-section title="Selected suites" width="full">
        <x-test-suites.select :select=false title="" :suites="$session->itemsGrouped" />
    </x-portal-section>

</x-layout>