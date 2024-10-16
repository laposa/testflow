<x-layout>
    <x-portal-section title="{{ $session->name }}">
        <button class="filled" onclick="window.print()">Print</button>
        <div class="session-environment">
            Environment: <span>{{ ucfirst($session->environment) }}</span>
        </div>
        <livewire:runs.list-runs :session="$session" />
    </x-portal-section>

    @if ($showLog)
    <div class="page-break"></div>

    @foreach ($session->runs as $run)
        <p>Run #{{ $run->id }} for {{ $run->service->displayName }}.</p>
        <p>Started at {{ $run->created_at }} and run for {{ $run->humanReadableDuration }}.</p>
        <p>
            <x-passed-failed passed="{{ $run->passed }}" failed="{{ $run->failed }}"
                             show-zero="true" />
        </p>
        <ul class="session-selected-suites">
            @foreach ($run->service->suites as $suite)
                <li>
                    {{ $suite->name }}
                </li>
                <ul>
                    @foreach ($suite->tests as $test)
                        @php($testCase = $run->parsedResults->getTestCase($test->name))
                        <li>
                            {{ $test->name }}
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
        @if (!$loop->last)
        <div class="page-break"></div>
        @endif

    @endforeach
    @endif
</x-layout>
