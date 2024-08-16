@php
    /** @var \App\Models\Session $session */
    /** @var \App\Models\SessionRun $run */
@endphp
<x-layout>
    <h2>{{ $session->name }}</h2>
    <p>Run No.{{ $run->id }} for {{ getTestServiceName($run->items[0]) }}</p>
    <section>
        <ul class="session-selected-suites">
            @foreach ($run->itemsGrouped as $path => $tests)
                @php($testSuite = $run->parsedResults->getTestSuite($tests[0]['suite_name']))
                <li>
                    <b>{{ $tests[0]['suite_name'] }}</b>
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
        <div x-data="{
            show: false
        }">
            <p>
                <a href="#" x-on:click="show = ! show">
                    <span x-text="show ? 'Hide' : 'Show'"></span> results in JUnit format
                </a>
            </p>
            <pre x-show="show">{{ $run->result_log }}</pre>
        </div>

    </section>

    <section x-data="{ show: false }">
        <p>
            <a href="#" x-on:click="show = ! show">
                <span x-text="show ? 'Hide' : 'Show'"></span> detailed run log
            </a>
        </p>

        <pre x-show="show">{{ $run->run_log }}</pre>

    </section>

</x-layout>
