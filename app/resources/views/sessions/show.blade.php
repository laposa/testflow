<x-layout>
    <h1>{{ $session->name }}</h1>
    <livewire:runs.list-runs :session="$session" />

    <x-portal-section title="Selected suites" width="full">
        <x-test-suites.select :select=false title="" :suites="$session->itemsGrouped" />
    </x-portal-section>
</x-layout>