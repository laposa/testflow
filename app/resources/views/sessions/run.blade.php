@php
    /** @var \App\Models\Session $session */
    /** @var \App\Models\SessionRun $run */
@endphp
<x-layout>
    <h2>{{ $session->name }}</h2>
    <p>Run #{{ $run->id }} for {{ getTestServiceName($run->items[0]) }}.</p>
    <section>
        <ul class="session-selected-suites">
            @foreach ($run->itemsGrouped as $path => $tests)
                @php($testSuite = $run->parsedResults->getTestSuite($tests[0]['suite_name']))
                <li>
                    {{ $tests[0]['suite_name'] }}
                </li>
                <ul>
                    @foreach ($tests as $test)
                        @php($testCase = $run->parsedResults->getTestCase($test['test_name']))
                        <li>
                            {{ $test['test_name'] }}
                            @if ($testCase)
                                <span title="Execution time {{ round($testCase['time'], 1) }}s"
                                    class="{{ $testCase['status'] }}"></span>
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
                <a class="button" href="#" x-on:click="show = ! show">
                    <span x-text="show ? 'Hide' : 'Show'"></span> results in JUnit format
                </a>
            </p>
            <pre x-show="show">{{ $run->result_log }}</pre>
        </div>

    </section>

    <section x-data="{ show: false }">
        <p>
            <a class="button" href="#" x-on:click="show = ! show">
                <span x-text="show ? 'Hide' : 'Show'"></span> detailed run log
            </a>
        </p>

        <pre x-show="show">{{ $run->run_log }}</pre>

    </section>

</x-layout>
