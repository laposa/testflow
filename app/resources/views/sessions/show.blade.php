<x-layout>
    <h1>{{ $session->title }}</h1>
    @foreach($session->workflow_runs as $suite => $runs)
        <x-portal-section title="{{ $suite }}" width="full" width="full">
            <x-runs.list :runs="$runs" />
        </x-portal-section>
    @endforeach


    <x-portal-section title="Test suites" width="full">
        <x-test-suites.list :suites="$session->data"/>
    </x-portal-section>

    <form action="{{ route('session.runs.store', ['session' => $session]) }}" method="POST">
        @csrf
        <button type="submit" class="button">Run selected tests</button>
    </form>

    <form action="{{ route('session.runs.store', ['session' => $session]) }}" method="POST">
        @csrf
        <button type="submit" class="button align-right">Run all tests</button>
    </form>
</x-layout>
