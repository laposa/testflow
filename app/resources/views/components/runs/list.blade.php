@props(['runs' => [], 'sessionId' => null])
<table class="runs-list">
    <thead>
        <tr>
            <th>Run ID</th>
            <th>Service</th>
            <th>Timestamp</th>
            <th>Status</th>
            <th>Results</th>
        </tr>
    </thead>
    <tbody>
        @foreach ($runs as $key => $run)
            <tr>
                <td>
                    @if ($run->run_log)
                        <a href="/sessions/{{ $sessionId }}/run/{{ $run['id'] }}"
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
            </tr>
        @endforeach
    </tbody>
</table>
