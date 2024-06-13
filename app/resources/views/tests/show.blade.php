<x-layout>
    <x-portal-section title="Recent runs for {{ $test->title }}" width="full">
        <x-runs.list :runs="$runs" />
    </x-portal-section>
</x-layout>
