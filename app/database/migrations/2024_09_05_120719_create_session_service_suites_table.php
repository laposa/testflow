<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration {
    /**
     * Run the migrations.
     */
    public function up(): void
    {
        Schema::create('test_session_service_suites', function (Blueprint $table) {
            $table->id();

            $table
                ->foreignId('service_id')
                ->constrained('test_session_services')
                ->onDelete('cascade')
                ->onUpdate('cascade');

            $table->string('name');
            $table->string('url');

            $table->timestamps();
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('session_service_suites');
    }
};