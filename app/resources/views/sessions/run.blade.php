@php
    /** @var \App\Models\Session $session */
    /** @var \App\Models\SessionServiceRun $run */
@endphp
<script type="text/javascript">

    // Open modal window
    function openModal() {
        var dialog = document.getElementById('test-dialog');
        dialog.querySelector('.content').innerText = 'Hello';
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
            var isInDialog = (rect.top <= event.clientY && event.clientY <= rect.top + rect.height &&
                rect.left <= event.clientX && event.clientX <= rect.left + rect.width);
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
                            <li class="test-file" onclick="openModal()" title="show file contents">
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

        <dialog id="test-dialog">
            <pre class="content">
            </pre>
            <form>
                <button formmethod="dialog">Close</button>
            </form>
        </dialog>
    </x-portal-section>
</x-layout>
