<x-layout>
    <h1>{{ $session->name }}</h1>
    @foreach($session->itemsGrouped as $path => $tests)
    <x-portal-section title="{{ getTestSuiteName($tests[0]) }}" width="full">
        <x-runs.list :runs="$tests[0]->runs" />
    </x-portal-section>
    @endforeach

    <div></div>
    <form action="{{ route('session.runs.store', ['session' => $session]) }}" method="POST">
        @csrf
        <button type="submit" class="button align-right">Run all tests</button>
    </form>

    <x-portal-section title="Selected suites" width="full">
        <x-test-suites.select :select=false title="" :suites="$session->itemsGrouped" />
    </x-portal-section>

</x-layout>