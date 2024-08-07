@php
    /** @var \App\Models\Session $session */
    /** @var \App\Models\SessionRun $run */
@endphp
<x-layout>
    <h1>{{ $session->title }}</h1>

    <x-portal-section title="Test suites">
        <ul class="session-selected-suites">
            @foreach ($run->itemsGrouped as $path => $tests)
                @php($testSuite = $run->parsedResults->getTestSuite($tests[0]['suite_name']))
                <li>
                    <b>{{ getTestSuiteName($tests[0]) }}</b>
                    @if ($testSuite)
                        <span class="run-time">{{ $testSuite['time'] }}s</span>
                    @else
                        <span class="not-found">not found</span>
                    @endif
                </li>
                <ul>
                    @foreach ($tests as $test)
                        @php($testCase = $run->parsedResults->getTestCase($test['test_name']))
                        <li>
                            {{ $test['test_name'] }}
                            @if ($testCase)
                                <span
                                    class="{{ $testCase['status'] }}">{{ $testCase['status'] }}</span>
                                <span class="run-time">{{ $testCase['time'] }}s</span>
                            @else
                                <span class="not-found">not found</span>
                            @endif
                        </li>
                    @endforeach
                </ul>
            @endforeach
        </ul>
    </x-portal-section>

    <x-portal-section title="Logs">
        <h3>Run Log</h3>
        <pre>{{ $run->run_log }}</pre>

        <h3>Results Log</h3>
        <pre>{{ $run->result_log }}</pre>
    </x-portal-section>

</x-layout>
