<x-layout>
    <h1>{{ $session->name }}</h1>
    @foreach($session->workflow_runs as $suite => $runs)
        <x-portal-section title="{{ $suite }}" width="full">
            <x-runs.list :runs="$runs" />
        </x-portal-section>
    @endforeach

    <div></div>
    <form action="{{ route('session.runs.store', ['session' => $session]) }}" method="POST">
        @csrf
        <button type="submit" class="button align-right">Run all tests</button>
    </form>

    <x-portal-section title="Selected suites" width="full">
        <x-test-suites.list :suites="$session->data"/>
    </x-portal-section>

</x-layout>
