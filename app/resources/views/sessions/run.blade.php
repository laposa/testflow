@php
    /** @var \App\Models\Session $session */
    /** @var \App\Models\SessionServiceRun $run */
@endphp
<script type="text/javascript">

    document.addEventListener('DOMContentLoaded', (event) => {
        var ansi_up = new AnsiUp;
        ansi_up._escape_html = false;

        document.querySelectorAll('pre').forEach((block) => {
            block.innerHTML = ansi_up.ansi_to_html(block.innerHTML);
        });
    });

</script>
<x-layout>
    <x-portal-section title="{{ $session->name }}">
        <div class="session-environment">
            Environment: <span>{{ ucfirst($session->environment) }}</span>
        </div>

        <p>Run #{{ $run->id }} for {{ $run->service->displayName }}.</p>
        <p>Started at {{ $run->created_at }} and run for {{ $run->humanReadableDuration }}.</p>
        <section>
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
            <div x-data="{
                show: false
            }">
                <p>
                    <a class="button" href="#" x-on:click.prevent="show = ! show">
                        <span x-text="show ? 'Hide' : 'Show'"></span> results in JUnit format
                    </a>
                </p>
                <pre x-show="show">{{ $run->result_log }}</pre>
            </div>

        </section>

        <section x-data="{ show: false }">
            <p>
                <a class="button" href="#" x-on:click.prevent="show = ! show">
                    <span x-text="show ? 'Hide' : 'Show'"></span> detailed run log
                </a>
            </p>

            <pre x-show="show">{{ $run->run_log }}</pre>

        </section>
    </x-portal-section>
</x-layout>
