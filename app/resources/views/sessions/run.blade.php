@php
    /** @var \App\Models\Session $session */
    /** @var \App\Models\SessionServiceRun $run */
@endphp
<script type="text/javascript">
    // Open modal window
    function openModal(testId) {
        var dialog = document.getElementById('test-dialog');
        dialog.querySelector('.content').style.display = 'none';
        dialog.querySelector('.loading').style.display = 'block';

        // make an ajax call and load the result into content
        const xhr = new XMLHttpRequest();
        xhr.open('GET', '/sessions/{{ $session->id }}/run/{{ $run->id }}/test/' + testId,
            true);
        xhr.onreadystatechange = function() {
            if (xhr.readyState === 4 && xhr.status === 200) {
                dialog.querySelector('.content').innerText = xhr.responseText;
                dialog.querySelector('.loading').style.display = 'none';
                dialog.querySelector('.content').style.display = 'block';
            }
        };
        xhr.send();

        dialog.showModal();
    }

    document.addEventListener('DOMContentLoaded', (event) => {
        // Log text formatter
        var ansi_up = new AnsiUp;
        ansi_up._escape_html = false;

        document.querySelectorAll('pre').forEach((block) => {
            block.innerHTML = ansi_up.ansi_to_html(block.innerHTML);
        });

        // Close dialog on backdrop click
        var dialog = document.getElementById('test-dialog');
        dialog.addEventListener('click', function(event) {
            var rect = dialog.getBoundingClientRect();
            var isInDialog = (
                rect.top <= event.clientY && event.clientY <= rect.top + rect
                .height &&
                rect.left <= event.clientX && event.clientX <= rect.left + rect
                .width);
            if (!isInDialog) {
                dialog.close();
            }
        });
    });
</script>
<x-layout>
    <x-portal-section title="{{ $session->name }}">
        <div class="session-environment">
            Environment: <span>{{ ucfirst($session->environment) }}</span>
        </div>

        <p>Run #{{ $run->id }} for {{ $run->service->displayName }}.</p>
        @if($run->finished_at)
            <p>Started at {{ $run->created_at }} and run for {{ $run->humanReadableDuration }}.</p>
        @else
            <p>Started at {{ $run->created_at }} and is still running ...</p>
        @endif
        <p>
            <x-passed-failed 
                passed="{{ $run->passed }}" 
                failed="{{ $run->failed }}"
                skipped="{{ $run->skipped }}"
                show-zero="true" 
            />
        </p>
        <section>
            <ul class="session-selected-suites">
                @foreach ($run->service->suites as $suite)
                    <li>
                        {{ $suite->name }}
                    </li>
                    <ul>
                        @foreach ($suite->tests as $test)
                            @php($testCase = $run->parsedResults->getTestCase($test->name))
                            <li class="test-file" onclick="openModal({{ $test->id }})"
                                title="show file contents">
                                {{ $test->name }}
                                @if ($testCase)
                                    <span title="Execution time {{ round($testCase['time'], 1) }}s"
                                        class="{{ $testCase['status'] }}"></span>
                                @else
                                    <span class="not-found">not found</span>
                                @endif
                            </li>
                            @if (array_key_exists("comment", $testCase ?? []) && $testCase['comment'])
                                <div class="comment">
                                    <div class="expand-content"></div>
                                    <div class="expandable-content">
                                        <i>{{ $testCase['comment'] }}</i>
                                    </div>
                                </div>
                            @endif
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

        <dialog id="test-dialog">
            <div class="loading">
                <x-loading-overlay relative="true"></x-loading-overlay>
            </div>
            <pre class="content"></pre>
            <form>
                <button formmethod="dialog">Close</button>
            </form>
        </dialog>
    </x-portal-section>
</x-layout>
