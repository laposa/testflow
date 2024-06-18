
@props(['runs' => []])
<table>
    <thead>
    <tr>
        <th>Run ID</th>
        <th>Test Name</th>
        <th>Timestamp</th>
        <th>Suite</th>
        <th>Session</th>
        <th>Status</th>
        <th>Conclusion</th>
    </tr>
    </thead>
    <tbody>
    @foreach($runs as $key => $run)
    <tr>
        <td>
            <a href="/sessions/1/run/{{ $run['id'] }}" title="Show logs">{{ $run['id'] }}</a>
        </td>
        <td>{{$run['name'] }}</td>
        <td>{{$run['run_started_at']}}</td>
        <td>{{$key}}</td>
        <td>{ TODO: SESSION }</td>
        <td>
            <span class="{{ $run['status'] }}">{{ $run['status'] }}</span>
        </td>

        <td>
           <span class="{{ $run['conclusion'] }}">{{ $run['conclusion'] }}</span>
        </td>
    </tr>
    @endforeach
    </tbody>
</table>
