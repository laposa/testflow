<x-layout>
    <h1>{{ $session->title }}</h1>
    <x-portal-section title="Recent runs" width="full" width="full">
        <x-runs.list :runs="$session->runs" />
    </x-portal-section>

    <x-portal-section title="Test suites" width="full">
        <x-test-suites.list :suites="$suites"/>
    </x-portal-section>

    <a href="#" class="button">Run selected tests</a>
    <a href="#" class="button align-right">Run all tests</a>
</x-layout>
