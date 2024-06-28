<x-layout>
    <h1>{{ $session->title }}</h1>

    <x-portal-section title="Test suites" width="full">
        <x-test-suites.list title="" :suites="$run->itemsGrouped" />
    </x-portal-section>

    <x-portal-section title="Logs" width="full">
        <pre>{{ $run->result_log }}</pre>
    </x-portal-section>

</x-layout>
