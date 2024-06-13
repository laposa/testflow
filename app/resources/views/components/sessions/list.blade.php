<div class="sessions">
    <div class="filter">
        <div class="search">
            <input type="text" placeholder="Search sessions" />
            <img src="/images/icons/search-icon.svg" alt="search" />
        </div>
        <input
            type="text"
            class="date"
            placeholder="Choose date"
            onfocus="(this.type='date')"
            onblur="(this.type='text')"
        />
        <select name="issuer" class="issuer">
            <option value="">All issuers</option>
            <option value="issuer1">Issuer 1</option>
            <option value="issuer2">Issuer 2</option>
            <option value="issuer3">Issuer 3</option>
        </select>
        <a href="/sessions/create">New session</a>
    </div>
</div>

<table>
    <thead>
    <tr>
        <th>Name</th>
        <th>Timestamp</th>
        <th>Issuer</th>
        <th>Results</th>
    </tr>
    </thead>
    <tbody>
        @foreach($sessions as $session)
            <tr>
                <td>{{$session->title}}</td>
                <td>{{$session->timestamp}}</td>
                <td>{{$session->issuer}}</td>
                <td><a href="/sessions/{{ $session->id }}">View</a></td>
            </tr>
        @endforeach
    </tbody>
</table>
