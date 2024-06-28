@props(['runs' => []])
<table>
    <thead>
        <tr>
            <th>Run ID</th>
            <th>Timestamp</th>
            <th>Status</th>
            <th>Results</th>
        </tr>
    </thead>
    <tbody>
        @foreach ($runs as $key => $run)
            <tr>
                <td>
                    @if ($run->result_log)
                        <a href="/sessions/1/run/{{ $run['id'] }}"
                            title="Show logs">{{ $run['id'] }}</a>
                    @else
                        {{ $run['id'] }}
                    @endif
                </td>
                <td>{{ $run['created_at'] }}</td>
                <td>
                    <span class="{{ $run['status'] }}">{{ $run['status'] }}</span>
                </td>
                <td>
                    <!--<span class="{{ $run['conclusion'] }}">{{ $run['conclusion'] }}</span>-->
                    <span class="pass">29 passed</span> and <span class="fail">2 failed</span>
                </td>
            </tr>
        @endforeach
    </tbody>
</table>
