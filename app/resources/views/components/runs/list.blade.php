
@props(['runs' => []])
<table>
    <thead>
    <tr>
        <th>Run ID</th>
        <th>Test Name</th>
        <th>Timestamp</th>
        <th>Suite</th>
<<<<<<< HEAD
        <th>Result</th>
=======
        <th>Session</th>
        <th>Status</th>
        <th>Conclusion</th>
        <th>Code</th>
        <th>Error</th>
>>>>>>> master
    </tr>
    </thead>
    <tbody>
    @foreach($runs as $key => $run)
    <tr>
        <td>
            <a href="/sessions/1/test/2/run/{{ $run['id'] }}">{{ $run['id'] }}</a>
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
        <td>
            @if ($run['status'])
                <a href="#">Show</a>
            @endif
        </td>
        <td>
{{--            @if ($run->error)--}}
{{--                <a href="#">Show</a>--}}
{{--            @endif--}}
        </td>
    </tr>
    @endforeach
    </tbody>
</table>
