<x-layout>
    <h1>{{ $session->title }}</h1>

    <x-portal-section title="Test suites">
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

    <x-portal-section title="Logs">
        <pre>{{ $run->result_log }}</pre>
    </x-portal-section>

</x-layout>
