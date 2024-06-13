
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
        <th>Result</th>
        <th>Code</th>
        <th>Error</th>
    </tr>
    </thead>
    <tbody>
    @foreach($runs as $run)
    <tr>
        <td>
            <a href="/sessions/1/test/2/run/{{ $run->id }}">{{ $run->id }}</a>
        </td>
        <td>{{$run->test->title }}</td>
        <td>{{$run->timestamp}}</td>
        <td>{{$run->suite}}</td>
        <td>{{$run->session}}</td>
        <td>
            @if ($run->result === "passed")
                <span class="pass">Passed</span>
            @else
                <span class="fail">Failed</span>
            @endif
        </td>
        <td>
            @if ($run->code)
                <a href="#">Show</a>
            @endif
        </td>
        <td>
            @if ($run->error)
                <a href="#">Show</a>
            @endif
        </td>
    </tr>
    @endforeach
    </tbody>
</table>
