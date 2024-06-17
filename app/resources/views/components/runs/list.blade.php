
@props(['runs' => []])
<div class="runs">
    <div class="filter">
        <div class="search">
            <input type="text" placeholder="Search runs" />
            <img src="/images/icons/search-icon.svg" alt="search" />
        </div>
        <input
            type="text"
            class="date"
            placeholder="Date from"
            onfocus="(this.type='date')"
            onblur="(this.type='text')"
        />
        <input
            type="text"
            class="date"
            placeholder="Date to"
            onfocus="(this.type='date')"
            onblur="(this.type='text')"
        />
        <select name="session" id="">
            <option value="">Session</option>
            <option value="0">Session 1</option>
            <option value="1">Session 2</option>
            <option value="2">Session 3</option>
        </select>
    </div>
</div>

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
        <th>Code</th>
        <th>Error</th>
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
