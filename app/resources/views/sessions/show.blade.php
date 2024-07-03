<x-layout>
    <x-portal-section title="{{ $session->name }}">
        <livewire:runs.list-runs :session="$session" />
    </x-portal-section>

    <x-portal-section title="Selected Tests">
        <ul class="session-selected-suites">
            @foreach ($session->itemsGrouped as $path => $tests)
                <li><b>{{ getTestSuiteName($tests[0]) }}</b></li>
                <ul>
                    @foreach ($tests as $test)
                        <li>{{ $test['test_name'] }}</li>
                    @endforeach
                </ul>
            @endforeach
        </ul>
    </x-portal-section>
</x-layout>
