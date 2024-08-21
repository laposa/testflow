@php
    /** @var \App\Models\Session $session */
@endphp

<div @if ($pollingEnabled) wire:poll.5s @endif>
    <table class="runs-list">
        <thead>
            <tr>
                <th>Run ID</th>
                <th>Service</th>
                <th>Started</th>
                <th>Status</th>
                <th>Results</th>
                <th></th>
            </tr>
        </thead>
        <tbody>
            @foreach ($session->runs as $key => $run)
                @php
                    /** @var \App\Models\SessionRun $run */
                @endphp
                <tr>
                    <td>
                        @if ($run->run_log)
                            <a href="/sessions/{{ $session->id }}/run/{{ $run['id'] }}"
                                title="Show logs">{{ $run['id'] }}</a>
                        @else
                            {{ $run['id'] }}
                        @endif
                    </td>
                    <td>{{ getTestServiceName($run->items[0]) }}</td>
                    <td>{{ $run['created_at'] }}</td>
                    <td>
                        <span class="{{ $run['status'] }}">{{ $run['status'] }}</span>
                    </td>
                    <td>
                        @if ($run->result_log)
                            <span class="pass">{{ $run->parsedResults->getTotalPassed() }}
                                passed</span> and <span
                                class="fail">{{ $run->parsedResults->getTotalFailures() }}
                                failed</span>
                        @endif
                    </td>
                    <td>
                        @if ($run->parsedResults->getTotalFailures() > 0 && $session->canRunTests())
                            <button wire:click="rerunRun({{ $run['id'] }})"
                                wire:loading.attr="disabled" class="filled">Rerun</button>
                        @endif
                    </td>
                </tr>
            @endforeach
        </tbody>
    </table>

    <button type="submit" wire:click="createRun" wire:loading.attr="disabled"
        @if (!$session->canRunTests()) disabled @endif class="filled">Run all tests</button>
</div>
