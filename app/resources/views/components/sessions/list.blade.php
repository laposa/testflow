<div class="sessions">
    <div class="filter">
        <div class="search">
            <input type="text" placeholder="Search sessions" />
            <img src="/images/icons/search-icon.svg" alt="search" />
        </div>
        <input type="text" class="date" placeholder="Choose date" onfocus="(this.type='date')"
            onblur="(this.type='text')" />
        <select name="issuer" class="issuer">
            <option value="">Any environment</option>
            <option value="issuer1">Prod</option>
            <option value="issuer2">Beta</option>
            <option value="issuer3">PreProd</option>
            <option value="issuer3">Dev</option>
        </select>
        <select name="issuer">
            <option value="">All issuers</option>
            <option value="issuer1">Martin Miksovsky</option>
            <option value="issuer2">Norbert Laposa</option>
            <option value="issuer3">Hugo Dvorak</option>
        </select>
        <a href="/sessions/create">New session</a>
    </div>
</div>

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Environment</th>
            <th>Timestamp</th>
            <th>Issuer</th>
        </tr>
    </thead>
    <tbody>
        @foreach ($sessions->reverse() as $session)
            <tr>
                <td><a href="/sessions/{{ $session->id }}">{{ $session->name }}</a></td>
                <td>Beta</td>
                <td>{{ $session->created_at->format('d M, Y h:m') }}</td>
                <td>{{ $session->issuer->name }}</td>
            </tr>
        @endforeach
    </tbody>
</table>
